package door

import (
	"fmt"
	"os/exec"
	"strings"
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
	for _, r := range cmd {
		if r < 32 || r > 126 {
			return fmt.Errorf("contains non-ASCII characters")
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
