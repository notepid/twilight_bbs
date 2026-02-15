package door

import (
	"fmt"
	"os/exec"
	"sync"
	"strings"
	"time"
)

// Launcher manages launching DOS doors via dosemu2.
type Launcher struct {
	DosemuPath string
	DriveCPath string
	TempDir    string
	Timeout    time.Duration // max door runtime; 0 defaults to 60 minutes

	mu    sync.Mutex
	inUse map[string]int
}

// NewLauncher creates a new door launcher.
func NewLauncher(dosemuPath, driveCPath, tempDir string) *Launcher {
	return &Launcher{
		DosemuPath: dosemuPath,
		DriveCPath: driveCPath,
		TempDir:    tempDir,
		inUse:      make(map[string]int),
	}
}

func normalizeDoorKey(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return name
}

// UsersInDoor returns how many users are currently running a door.
// The door is identified by its configured name.
func (l *Launcher) UsersInDoor(doorName string) int {
	key := normalizeDoorKey(doorName)
	if key == "" {
		return 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inUse[key]
}

// reserveDoor increments usage for the given door and returns a release function.
// If the door is not multi-user and is already in use, an error is returned.
func (l *Launcher) reserveDoor(cfg *Config) (func(), error) {
	if cfg == nil {
		return func() {}, fmt.Errorf("missing door config")
	}

	key := normalizeDoorKey(cfg.Name)
	if key == "" {
		return func() {}, fmt.Errorf("door config missing 'name'")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.inUse[key]
	if !cfg.MultiUser && current > 0 {
		return func() {}, fmt.Errorf("door '%s' is currently in use (%d user(s))", cfg.Name, current)
	}

	l.inUse[key] = current + 1

	released := false
	release := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true
		if c := l.inUse[key]; c <= 1 {
			delete(l.inUse, key)
		} else {
			l.inUse[key] = c - 1
		}
	}

	return release, nil
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
	if strings.ContainsAny(cmd, "\x00\r\n") {
		return fmt.Errorf("contains control characters")
	}
	if strings.ContainsAny(cmd, "&|;><`$") {
		return fmt.Errorf("contains shell metacharacters")
	}
	// Strip placeholders before checking for non-printable chars,
	// since {NODE} and {DROP} use braces which are valid.
	stripped := strings.NewReplacer("{NODE}", "", "{DROP}", "").Replace(cmd)
	for _, r := range stripped {
		if r < 32 || r > 126 {
			return fmt.Errorf("contains non-printable characters")
		}
	}
	return nil
}

// expandDoorCommand replaces placeholders in a door command string.
//
//	{NODE} → node number
//	{DROP} → DOS path to the drop file directory (e.g. C:\BBS\DROP\NODE1)
func expandDoorCommand(command string, nodeID int, dosDropDir string) string {
	r := strings.NewReplacer(
		"{NODE}", fmt.Sprintf("%d", nodeID),
		"{DROP}", dosDropDir,
	)
	return r.Replace(command)
}
