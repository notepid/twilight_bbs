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
)

// Launcher manages launching DOS doors via dosemu2.
type Launcher struct {
	DosemuPath string
	DriveCPath string
	TempDir    string
}

// NewLauncher creates a new door launcher.
func NewLauncher(dosemuPath, driveCPath, tempDir string) *Launcher {
	return &Launcher{
		DosemuPath: dosemuPath,
		DriveCPath: driveCPath,
		TempDir:    tempDir,
	}
}

// Launch starts a DOS door, bridging I/O between the user's terminal and
// the dosemu2 process. This blocks until the door exits.
func (l *Launcher) Launch(session *Session, stdin io.Reader, stdout io.Writer) error {
	// Create a temporary directory for this door session
	sessionDir := filepath.Join(l.TempDir, fmt.Sprintf("node%d", session.NodeID))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	defer os.RemoveAll(sessionDir)

	// Write drop file
	dropPath, err := WriteDropFile(sessionDir, session)
	if err != nil {
		return fmt.Errorf("write drop file: %w", err)
	}
	session.DropFilePath = dropPath

	log.Printf("Door: launching %s for node %d (drop file: %s)",
		session.DoorConfig.Name, session.NodeID, dropPath)

	if err := validateDoorCommand(session.DoorConfig.Command); err != nil {
		return fmt.Errorf("invalid door command: %w", err)
	}

	// Build dosemu2 command
	// dosemu2 can be run in dumb terminal mode with -t
	// The door executable runs inside the DOS environment
	cmd := exec.Command(l.DosemuPath,
		"-t",                         // dumb terminal mode
		"-E", session.DoorConfig.Command, // execute command
		"--Fimagedir", l.DriveCPath,  // set drive C
	)

	cmd.Dir = sessionDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DOORWAY_NODE=%d", session.NodeID),
		fmt.Sprintf("DOORWAY_DROP=%s", dropPath),
	)

	// Set up I/O bridging
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start dosemu: %w", err)
	}

	// Bridge I/O with goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	stdinDone := make(chan struct{})

	// User -> Door (stdin)
	go func() {
		defer close(stdinDone)
		defer cmdStdin.Close()
		io.Copy(cmdStdin, stdin)
	}()

	// Door -> User (stdout)
	go func() {
		defer wg.Done()
		io.Copy(stdout, cmdStdout)
	}()

	// Wait for the process to exit
	err = cmd.Wait()
	wg.Wait()

	// Don't hang the session if stdin never unblocks after the door exits.
	select {
	case <-stdinDone:
	case <-time.After(200 * time.Millisecond):
		log.Printf("Door: stdin copy still running after exit (node %d)", session.NodeID)
	}

	if err != nil {
		log.Printf("Door %s exited with error: %v", session.DoorConfig.Name, err)
		// Don't return error - door exit codes are often non-zero
	} else {
		log.Printf("Door %s exited normally for node %d",
			session.DoorConfig.Name, session.NodeID)
	}

	return nil
}

// Available checks if dosemu2 is installed and accessible.
func (l *Launcher) Available() bool {
	_, err := exec.LookPath(l.DosemuPath)
	return err == nil
}

func validateDoorCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return fmt.Errorf("empty")
	}
	if len(cmd) > 256 {
		return fmt.Errorf("too long")
	}
	// Disallow control characters and common command separators.
	if strings.ContainsAny(cmd, "\x00\r\n") {
		return fmt.Errorf("contains control characters")
	}
	if strings.ContainsAny(cmd, "&|;><`$") {
		return fmt.Errorf("contains shell metacharacters")
	}
	for _, r := range cmd {
		if r < 32 || r > 126 {
			return fmt.Errorf("contains non-ASCII characters")
		}
	}
	return nil
}
