//go:build linux

package door

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/creack/pty"
)

// resolveDosemuBinary finds the actual dosemu2 binary. The /usr/bin/dosemu path
// is typically a shell wrapper script; the real binary is at a well-known libexec
// path. Calling the binary directly avoids shell quoting issues and gives us
// direct control over all flags.
func resolveDosemuBinary(wrapperPath string) string {
	// Try well-known binary locations first.
	candidates := []string{
		"/usr/libexec/dosemu2/dosemu2.bin",
		"/usr/lib/dosemu2/dosemu2.bin",
		"/usr/local/libexec/dosemu2/dosemu2.bin",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Fall back to the wrapper path (will still work, just with banner).
	return wrapperPath
}

// Launch starts a DOS door via dosemu2 using a PTY for I/O.
//
// Architecture (matches enigma-bbs "stdio" approach for DOSEMU):
//
//	BBS terminal (stdin/stdout) <──> PTY master fd <──> dosemu2 process
//
// dosemu2 is configured with $_com1 = "virtual", which maps DOS COM1
// to its stdio. The PTY provides the terminal emulation layer that
// dosemu2 expects. BNU FOSSIL driver inside DOS provides the FOSSIL
// API that door games use to talk to COM1.
func (l *Launcher) Launch(session *Session, stdin io.Reader, stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	resolvePath := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(wd, p)
	}

	// Validate the door command before proceeding.
	if err := validateDoorCommand(session.DoorConfig.Command); err != nil {
		return fmt.Errorf("invalid door command: %w", err)
	}

	// --- Session directory (holds .dosemu/dosemurc) ----------------------
	tempDir := resolvePath(l.TempDir)
	sessionDir := filepath.Join(tempDir, fmt.Sprintf("node%d", session.NodeID))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	defer os.RemoveAll(sessionDir)

	// Resolve drive C path
	driveC := resolvePath(l.DriveCPath)

	// DOSEMU's 'virtual' COM driver maps DOS COM1 to stdio.
	session.ComPort = 1

	// --- Write drop file into a DOS-visible directory on drive C ---------
	dropDir := filepath.Join(driveC, "NODES", fmt.Sprintf("TEMP%d", session.NodeID))
	dropPath, err := WriteDropFile(dropDir, session)
	if err != nil {
		return fmt.Errorf("write drop file: %w", err)
	}
	session.DropFilePath = dropPath
	defer os.RemoveAll(dropDir)

	// --- Generate wrapper batch file ------------------------------------
	dosDropDir := fmt.Sprintf(`C:\NODES\TEMP%d`, session.NodeID)
	doorCmd := expandDoorCommand(session.DoorConfig.Command, session.NodeID, dosDropDir)

	wrapperBat := filepath.Join(dropDir, "RUN.BAT")
	batLines := []string{
		"@ECHO OFF",
		"CLS",
		`C:\BNU\BNU.COM /L0:57600,8N1 /F`,
		doorCmd,
		"EXITEMU",
		"",
	}
	batContent := strings.Join(batLines, "\r\n")
	if err := os.WriteFile(wrapperBat, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("write wrapper bat: %w", err)
	}
	finalDoorCmd := fmt.Sprintf(`C:\NODES\TEMP%d\RUN.BAT`, session.NodeID)

	// --- Generate dosemurc ----------------------------------------------
	dosemuLocalDir := filepath.Join(sessionDir, ".dosemu")
	if err := os.MkdirAll(dosemuLocalDir, 0755); err != nil {
		return fmt.Errorf("create dosemu local dir: %w", err)
	}
	dosemurcPath := filepath.Join(dosemuLocalDir, "dosemurc")
	dosemurc := strings.Join([]string{
		fmt.Sprintf(`$_hdimage = "%s +1"`, driveC),
		`$_cpu = "80486"`,
		`$_cpu_emu = "vm86"`,
		`$_layout = "us"`,
		`$_speaker = "off"`,
		`$_sound = (off)`,
		`$_hogthreshold = (1)`,
		`$_external_char_set = "cp437"`,
		`$_internal_char_set = "cp437"`,
		`$_com1 = "virtual"`,
		`$_rawkeyboard = (0)`,
		`$_term_updfreq = (8)`,
		`$_term_color = (on)`,
		`$_lpt1 = ""`,
		"",
	}, "\n")
	if err := os.WriteFile(dosemurcPath, []byte(dosemurc), 0644); err != nil {
		return fmt.Errorf("write dosemu rc: %w", err)
	}

	// --- Resolve the dosemu2 binary path --------------------------------
	// We call the binary directly instead of the /usr/bin/dosemu shell wrapper.
	// This avoids the wrapper's eval/quoting issues and gives us direct control
	// over all flags (-q for quiet boot, -f for config, -E for command, etc.).
	dosemuBin := resolveDosemuBinary(l.DosemuPath)

	// --- Build stderr log path ------------------------------------------
	stderrLog := filepath.Join(dosemuLocalDir, "boot.log")

	// --- Determine terminal dimensions ----------------------------------
	termW := uint16(session.TermWidth)
	if termW == 0 {
		termW = 80
	}
	termH := uint16(session.TermHeight)
	if termH == 0 {
		termH = 25
	}

	// --- Spawn dosemu2 in a PTY -----------------------------------------
	timeout := l.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dosemuBin,
		"-f", dosemurcPath,    // explicit config file
		"-o", stderrLog,       // debug/error log file
		"-td",                 // dumb terminal mode (text output on stdout)
		"-ks",                 // keyboard in stdio mode
		"-q",                  // quiet: suppress boot banner
		"-E", finalDoorCmd,    // execute DOS command and exit
	)
	cmd.Dir = driveC
	// HOME must still point to sessionDir so dosemu finds ~/.dosemu/ for
	// internal state (drive setup, etc.), but we pass -f explicitly.
	cmd.Env = []string{
		"HOME=" + sessionDir,
		"TERM=dumb",
		"LANG=en_US.UTF-8",
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	}

	winSize := &pty.Winsize{
		Rows: termH,
		Cols: termW,
	}

	log.Printf("[door] Node %d: spawning dosemu2 — bin=%s door=%s pty=%dx%d",
		session.NodeID, dosemuBin, session.DoorConfig.Name, termW, termH)

	// We call the dosemu2 binary directly (not the shell wrapper), so
	// pty.StartWithSize will set stdin/stdout/stderr to the PTY slave.
	// Errors go to the -o log file.

	ptmx, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return fmt.Errorf("pty start dosemu: %w", err)
	}
	defer ptmx.Close()

	// --- Bridge I/O: BBS terminal <-> PTY master ------------------------
	//
	// Two goroutines bridge the data:
	//   1. stdin → ptmx  (user keystrokes → door)
	//   2. ptmx → stdout (door output → user terminal)
	//
	// When the door process exits we close ptmx, which unblocks goroutine 2.
	// Goroutine 1 may remain blocked on stdin.Read(); it uses a cancel channel
	// to exit cleanly on its next read cycle. One keystroke may be consumed
	// and discarded during cleanup — this is expected BBS door behaviour and
	// is absorbed by the node:pause() that follows in the menu script.

	cancelInput := make(chan struct{})
	inputDone := make(chan struct{})

	// User input → dosemu (stdin → ptmx)
	go func() {
		defer close(inputDone)
		buf := make([]byte, 4096)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				select {
				case <-cancelInput:
					return
				default:
				}
				if _, werr := ptmx.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// dosemu output → user terminal (ptmx → stdout)
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		io.Copy(stdout, ptmx)
	}()

	// Wait for dosemu to exit
	waitErr := cmd.Wait()

	// Signal the input goroutine to stop forwarding
	close(cancelInput)

	// Give the output goroutine time to drain any remaining data from the PTY
	// buffer. When dosemu exits, the slave side of the PTY closes, but there
	// may still be data in the kernel PTY buffer that hasn't been read yet.
	// The output goroutine's io.Copy will read until it gets an I/O error
	// from the closed slave — we wait for that rather than closing early.
	select {
	case <-outputDone:
		// Output goroutine finished reading all data
	case <-time.After(2 * time.Second):
		// Safety timeout — close ptmx to force the goroutine to exit
		log.Printf("[door] Node %d: output drain timed out, closing PTY", session.NodeID)
		ptmx.Close()
		<-outputDone
	}

	// Close the PTY master (may already be closed by timeout above)
	ptmx.Close()

	// Give the input goroutine a brief moment to exit if it already has data
	select {
	case <-inputDone:
	case <-time.After(100 * time.Millisecond):
		// Input goroutine is blocked on stdin.Read(); it will exit on the
		// next keystroke when it sees cancelInput is closed.
	}

	if waitErr != nil && ctx.Err() == context.DeadlineExceeded {
		log.Printf("[door] Node %d: door timed out after %v", session.NodeID, timeout)
		return fmt.Errorf("door timed out after %v", timeout)
	}

	if waitErr != nil {
		log.Printf("[door] Node %d: dosemu2 exited with: %v", session.NodeID, waitErr)
	}

	log.Printf("[door] Node %d: door session ended", session.NodeID)
	return nil
}
