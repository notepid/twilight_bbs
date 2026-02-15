package transfer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// defaultTimeout is the maximum duration for a single transfer.
const defaultTimeout = 30 * time.Minute

// Send initiates a ZMODEM-8K download (BBS → user) of the given files
// using SEXYZ via the supplied raw ReadWriter.
//
// If isTelnet is true, the -telnet flag is passed to SEXYZ so it handles
// IAC escaping/filtering itself.
func (c *Config) Send(rw io.ReadWriter, isTelnet bool, filePaths ...string) (*Result, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files to send")
	}

	// Convert all paths to absolute and verify they exist.
	absPaths := make([]string, len(filePaths))
	for i, fp := range filePaths {
		abs, err := filepath.Abs(fp)
		if err != nil {
			return nil, formatError("resolve file path", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return nil, formatError("file not found", err)
		}
		absPaths[i] = abs
	}

	// Build SEXYZ command:
	//   sexyz [-telnet] -y -8 sz <file1> [file2 ...]
	//
	// "sz" = send via ZMODEM-8K.
	args := buildArgs(isTelnet, "sz", absPaths)

	log.Printf("[transfer] SEND starting: sexyz %v", args)

	result, err := c.run(rw, args, "")
	if err != nil {
		return nil, formatError("send failed", err)
	}

	// Populate the result with the files we sent.
	for _, fp := range absPaths {
		info, _ := os.Stat(fp)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		result.Files = append(result.Files, TransferredFile{
			Name: filepath.Base(fp),
			Size: size,
		})
	}

	log.Printf("[transfer] SEND complete: %d file(s)", len(result.Files))
	return result, nil
}

// Receive initiates a ZMODEM upload (user → BBS) into the given directory
// using SEXYZ via the supplied raw ReadWriter. Returns information about
// the file(s) received.
func (c *Config) Receive(rw io.ReadWriter, isTelnet bool, uploadDir string) (*Result, error) {
	// Convert to absolute path so SEXYZ resolves it unambiguously.
	absDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return nil, formatError("resolve upload dir", err)
	}

	// SEXYZ concatenates the directory path with the filename directly,
	// so it MUST end with a path separator.
	if !strings.HasSuffix(absDir, "/") {
		absDir += "/"
	}

	// Ensure the upload directory exists.
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, formatError("create upload dir", err)
	}

	// Snapshot existing files before the transfer so we can detect new ones.
	before, err := snapshotDir(absDir)
	if err != nil {
		return nil, formatError("snapshot dir", err)
	}

	// Build SEXYZ command:
	//   sexyz [-telnet] -y -8 rz <absDir>
	//
	// "rz" = receive via ZMODEM. -y allows overwriting existing files.
	// The absolute directory path tells SEXYZ where to save received files.
	args := buildArgs(isTelnet, "rz", nil)
	args = append(args, absDir)

	log.Printf("[transfer] RECEIVE starting into %s: sexyz %v", absDir, args)

	// Don't set workDir — the absolute path in args is sufficient.
	_, runErr := c.run(rw, args, "")

	// Even if SEXYZ exits non-zero (e.g. user cancelled), check what arrived.
	after, err := snapshotDir(absDir)
	if err != nil {
		return nil, formatError("snapshot dir after receive", err)
	}

	result := &Result{}
	for name, size := range after {
		if _, existed := before[name]; !existed {
			result.Files = append(result.Files, TransferredFile{
				Name: name,
				Size: size,
			})
		}
	}

	if runErr != nil && len(result.Files) == 0 {
		return nil, formatError("receive failed", runErr)
	}

	log.Printf("[transfer] RECEIVE complete: %d new file(s)", len(result.Files))
	return result, nil
}

// run spawns SEXYZ with the given arguments, bridges I/O between the
// raw connection and the process, and waits for completion.
func (c *Config) run(rw io.ReadWriter, args []string, workDir string) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Create a Unix socketpair. SEXYZ in stdio mode uses fd 0 (stdin) and
	// fd 1 (stdout) for reading/writing. Some builds of SEXYZ also use
	// select() on these fds, which works with sockets but not with Go's
	// internal pipes. A socketpair gives SEXYZ a real bidirectional socket
	// on fds 0 and 1.
	parentConn, childFile, err := createSocketPair()
	if err != nil {
		return nil, fmt.Errorf("create socketpair: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.SexyzPath, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Stdin = childFile
	cmd.Stdout = childFile
	// Capture stderr so we can return actionable errors.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		childFile.Close()
		parentConn.Close()
		return nil, fmt.Errorf("start sexyz: %w", err)
	}
	// Child fd is now inherited by SEXYZ; close our copy so EOF propagates.
	childFile.Close()

	log.Printf("[transfer] sexyz started (pid %d): %s %s", cmd.Process.Pid, c.SexyzPath, strings.Join(args, " "))

	// Bridge I/O: remote client <-> SEXYZ process (via socketpair)
	var inputBytes int64
	var outputBytes int64

	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		n, err := io.Copy(parentConn, rw)
		atomic.StoreInt64(&inputBytes, n)
		log.Printf("[transfer] input goroutine done: %d bytes client→sexyz, err=%v", n, err)
	}()

	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		n, err := io.Copy(rw, parentConn)
		atomic.StoreInt64(&outputBytes, n)
		log.Printf("[transfer] output goroutine done: %d bytes sexyz→client, err=%v", n, err)
	}()

	// Wait for SEXYZ to exit.
	waitErr := cmd.Wait()

	log.Printf("[transfer] sexyz exited: err=%v, input=%d bytes, output=%d bytes",
		waitErr, atomic.LoadInt64(&inputBytes), atomic.LoadInt64(&outputBytes))

	// Close our end of the socketpair to unblock the copy goroutines.
	parentConn.Close()

	// Always log stderr for debugging.
	if stderr.Len() > 0 {
		log.Printf("[transfer] sexyz stderr:\n%s", stderr.String())
	}

	// Wait for the output goroutine to drain any remaining data.
	select {
	case <-outputDone:
	case <-time.After(3 * time.Second):
		log.Printf("[transfer] output drain timed out")
	}

	// The input goroutine may be blocked on rw.Read(); it will unblock
	// when the connection sends more data or closes. We don't wait.

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("transfer timed out after %v", defaultTimeout)
	}

	result := &Result{}
	if waitErr != nil {
		if stderr.Len() > 0 {
			waitErr = fmt.Errorf("%w: %s", waitErr, strings.TrimSpace(stderr.String()))
		}
		// Non-zero exit from SEXYZ. This can happen if the user cancels
		// the transfer. We return the error but the caller may still check
		// for partially received files.
		result.Error = waitErr
		log.Printf("[transfer] sexyz exited with: %v", waitErr)
	}

	return result, waitErr
}

// buildArgs constructs the SEXYZ command-line arguments.
func buildArgs(isTelnet bool, mode string, files []string) []string {
	var args []string
	if isTelnet {
		// In stdio mode, Telnet handling is disabled by default.
		args = append(args, "-telnet")
	}
	// -y allows overwriting existing files. -8 selects ZMODEM-8K (ZedZap).
	args = append(args, "-y", "-8", mode)
	args = append(args, files...)
	return args
}

// snapshotDir returns a map of filename → size for all regular files
// in the given directory (non-recursive).
func snapshotDir(dir string) (map[string]int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		m[e.Name()] = info.Size()
	}
	return m, nil
}
