//go:build linux

package door

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Launch starts a DOS door, bridging I/O between the user's terminal and
// the dosemu2 process via a PTY. This blocks until the door exits.
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

	// Create a temporary directory for this door session
	tempDir := resolvePath(l.TempDir)
	sessionDir := filepath.Join(tempDir, fmt.Sprintf("node%d", session.NodeID))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	defer os.RemoveAll(sessionDir)

	// Resolve drive C path
	driveC := resolvePath(l.DriveCPath)

	// Write drop file into a DOS-visible location on drive C
	dropDir := filepath.Join(driveC, "BBS", "DROP", fmt.Sprintf("NODE%d", session.NodeID))
	dropPath, err := WriteDropFile(dropDir, session)
	if err != nil {
		return fmt.Errorf("write drop file: %w", err)
	}
	session.DropFilePath = dropPath
	defer os.RemoveAll(dropDir)

	log.Printf("Door: launching %s for node %d (drop file: %s)",
		session.DoorConfig.Name, session.NodeID, dropPath)

	// Expand placeholders in the door command
	dosDropDir := fmt.Sprintf("C:\\BBS\\DROP\\NODE%d", session.NodeID)
	command := expandDoorCommand(session.DoorConfig.Command, session.NodeID, dosDropDir)

	if err := validateDoorCommand(command); err != nil {
		return fmt.Errorf("invalid door command: %w", err)
	}

	// Use a per-session DOSEMU local dir
	dosemuLocalDir := filepath.Join(sessionDir, ".dosemu")
	if err := os.MkdirAll(dosemuLocalDir, 0755); err != nil {
		return fmt.Errorf("create dosemu local dir: %w", err)
	}

	// Write dosemurc with CP437, virtual COM1, and other door-friendly settings
	dosemurcPath := filepath.Join(dosemuLocalDir, "dosemurc")
	dosemurc := strings.Join([]string{
		`$_cpu_vm = "emulated"`,
		`$_cpu_vm_dpmi = "emulated"`,
		`$_layout = "us"`,
		`$_speaker = "off"`,
		`$_sound = (off)`,
		`$_external_char_set = "cp437"`,
		`$_internal_char_set = "cp437"`,
		`$_com1 = "virtual"`,
		`$_rawkeyboard = (0)`,
		"",
	}, "\n")
	if err := os.WriteFile(dosemurcPath, []byte(dosemurc), 0644); err != nil {
		return fmt.Errorf("write dosemu rc: %w", err)
	}

	cmd := exec.Command(l.DosemuPath,
		"-t",
		"-E", command,
		"--Flocal_dir", dosemuLocalDir,
		"--Fdrive_c", driveC,
	)

	cmd.Dir = sessionDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DOORWAY_NODE=%d", session.NodeID),
		fmt.Sprintf("DOORWAY_DROP=%s", dropPath),
	)

	// Redirect stderr to a log file in the session directory
	stderrLogPath := filepath.Join(sessionDir, "dosemu_stderr.log")
	stderrFile, err := os.Create(stderrLogPath)
	if err != nil {
		return fmt.Errorf("create stderr log: %w", err)
	}
	defer stderrFile.Close()
	cmd.Stderr = stderrFile

	// Determine PTY window size
	cols := uint16(session.TermWidth)
	if cols == 0 {
		cols = 80
	}
	rows := uint16(session.TermHeight)
	if rows == 0 {
		rows = 25
	}

	// Start the process under a PTY so dosemu2 sees a real terminal
	winSize := &pty.Winsize{
		Cols: cols,
		Rows: rows,
	}
	ptmx, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return fmt.Errorf("start dosemu with pty: %w", err)
	}
	defer ptmx.Close()

	// Bridge I/O between user terminal and PTY
	var wg sync.WaitGroup
	wg.Add(1)

	stdinDone := make(chan struct{})

	// User -> Door (stdin to PTY)
	go func() {
		defer close(stdinDone)
		io.Copy(ptmx, stdin)
	}()

	// Door -> User (PTY to stdout)
	go func() {
		defer wg.Done()
		io.Copy(stdout, ptmx)
	}()

	// Wait for the process to exit
	err = cmd.Wait()
	wg.Wait()

	// Don't hang if stdin copy never unblocks
	select {
	case <-stdinDone:
	case <-time.After(200 * time.Millisecond):
		log.Printf("Door: stdin copy still running after exit (node %d)", session.NodeID)
	}

	// Log any stderr output
	if info, statErr := os.Stat(stderrLogPath); statErr == nil && info.Size() > 0 {
		if data, readErr := os.ReadFile(stderrLogPath); readErr == nil {
			log.Printf("Door %s stderr (node %d):\n%s", session.DoorConfig.Name, session.NodeID, string(data))
		}
	}

	if err != nil {
		log.Printf("Door %s exited with error: %v", session.DoorConfig.Name, err)
	} else {
		log.Printf("Door %s exited normally for node %d",
			session.DoorConfig.Name, session.NodeID)
	}

	return nil
}
