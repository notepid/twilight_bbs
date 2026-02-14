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
	"golang.org/x/sys/unix"
)

// Launch starts a DOS door, bridging I/O between the user's terminal and
// the dosemu2 process. FOSSIL-based doors communicate via a virtual COM1
// port backed by a PTY pair; dosemu2's built-in FOSSIL.COM driver connects
// the DOS INT 14h interface to this port. This blocks until the door exits.
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

	// --- COM1 PTY pair --------------------------------------------------
	// Create a PTY pair for the virtual serial port. The slave device path
	// is given to dosemu2 as $_com1; the master is bridged to the BBS user.
	comMaster, comSlave, err := pty.Open()
	if err != nil {
		return fmt.Errorf("create COM1 pty: %w", err)
	}
	comSlavePath := comSlave.Name()

	// Set raw mode on the COM master so the kernel PTY layer does not
	// interfere with the binary data stream (no echo, no CR/LF mapping,
	// no signal characters).
	if err := makeRaw(int(comMaster.Fd())); err != nil {
		comMaster.Close()
		comSlave.Close()
		return fmt.Errorf("set COM1 pty raw mode: %w", err)
	}

	// Close the slave fd after dosemu2 starts — dosemu2 will re-open the
	// device by path. Keeping our reference open would prevent EOF detection.
	defer comMaster.Close()

	// --- Wrapper batch file ---------------------------------------------
	// Create a batch file on drive C that loads dosemu2's built-in FOSSIL
	// driver (Z:\FOSSIL.COM) and then runs the actual door command.
	wrapperDir := filepath.Join(driveC, "BBS", "TEMP", fmt.Sprintf("NODE%d", session.NodeID))
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		comSlave.Close()
		return fmt.Errorf("create wrapper dir: %w", err)
	}
	defer os.RemoveAll(wrapperDir)

	wrapperBat := filepath.Join(wrapperDir, "RUN.BAT")
	batContent := "@ECHO OFF\r\nZ:\\FOSSIL.COM\r\n" + command + "\r\nEXIT\r\n"
	if err := os.WriteFile(wrapperBat, []byte(batContent), 0644); err != nil {
		comSlave.Close()
		return fmt.Errorf("write wrapper bat: %w", err)
	}
	dosBatPath := fmt.Sprintf("C:\\BBS\\TEMP\\NODE%d\\RUN.BAT", session.NodeID)

	// --- dosemurc -------------------------------------------------------
	dosemuLocalDir := filepath.Join(sessionDir, ".dosemu")
	if err := os.MkdirAll(dosemuLocalDir, 0755); err != nil {
		comSlave.Close()
		return fmt.Errorf("create dosemu local dir: %w", err)
	}

	dosemurcPath := filepath.Join(dosemuLocalDir, "dosemurc")
	dosemurc := strings.Join([]string{
		`$_cpu_vm = "emulated"`,
		`$_cpu_vm_dpmi = "emulated"`,
		`$_layout = "us"`,
		`$_speaker = "off"`,
		`$_sound = (off)`,
		`$_external_char_set = "cp437"`,
		`$_internal_char_set = "cp437"`,
		fmt.Sprintf(`$_com1 = "%s"`, comSlavePath),
		`$_rawkeyboard = (0)`,
		"",
	}, "\n")
	if err := os.WriteFile(dosemurcPath, []byte(dosemurc), 0644); err != nil {
		comSlave.Close()
		return fmt.Errorf("write dosemu rc: %w", err)
	}

	log.Printf("Door: COM1 pty slave=%s, wrapper=%s", comSlavePath, dosBatPath)

	// --- Launch dosemu2 -------------------------------------------------
	cmd := exec.Command(l.DosemuPath,
		"-t",
		"-E", dosBatPath,
		"--Flocal_dir", dosemuLocalDir,
		"--Fdrive_c", driveC,
	)

	cmd.Dir = sessionDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DOORWAY_NODE=%d", session.NodeID),
		fmt.Sprintf("DOORWAY_DROP=%s", dropPath),
	)

	// Redirect stderr to a log file
	stderrLogPath := filepath.Join(sessionDir, "dosemu_stderr.log")
	stderrFile, err := os.Create(stderrLogPath)
	if err != nil {
		comSlave.Close()
		return fmt.Errorf("create stderr log: %w", err)
	}
	defer stderrFile.Close()
	cmd.Stderr = stderrFile

	// dosemu2 -t needs a real terminal for its console. We provide one via
	// a PTY but we do NOT bridge it to the user — only the COM1 channel
	// carries the door's actual I/O.
	consolePtmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 25})
	if err != nil {
		comSlave.Close()
		return fmt.Errorf("start dosemu with pty: %w", err)
	}
	defer consolePtmx.Close()

	// Close our copy of the slave fd now that dosemu2 is running.
	comSlave.Close()

	// --- I/O bridging ---------------------------------------------------
	var wg sync.WaitGroup
	wg.Add(2)

	stdinDone := make(chan struct{})

	// Drain the console PTY (DOS prompt noise, FOSSIL.COM banner, etc.).
	go func() {
		defer wg.Done()
		io.Copy(io.Discard, consolePtmx)
	}()

	// User → Door: forward user input to COM1 master (→ FOSSIL → door)
	go func() {
		defer close(stdinDone)
		io.Copy(comMaster, stdin)
	}()

	// Door → User: forward COM1 master output (door → FOSSIL →) to the user
	go func() {
		defer wg.Done()
		io.Copy(stdout, comMaster)
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

// makeRaw puts a file descriptor into raw mode (no echo, no signal
// processing, no CR/LF translation). This is equivalent to cfmakeraw(3).
func makeRaw(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	termios.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
