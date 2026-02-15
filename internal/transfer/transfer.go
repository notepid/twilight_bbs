package transfer

import (
	"fmt"
	"os"
)

// Config holds file transfer protocol settings.
type Config struct {
	SexyzPath     string         // path to the sexyz binary
	PathValidator *PathValidator // optional validator for file paths
}

// TransferredFile describes a single file that was transferred.
type TransferredFile struct {
	Name string
	Size int64
}

// Result describes the outcome of a file transfer operation.
type Result struct {
	Files []TransferredFile
	Error error
}

// Available checks whether the SEXYZ binary exists and is executable.
func (c *Config) Available() bool {
	info, err := os.Stat(c.SexyzPath)
	if err != nil {
		return false
	}
	// On Linux, check that it's not a directory and has some size.
	return !info.IsDir() && info.Size() > 0
}

// formatError creates a user-friendly error message.
func formatError(action string, err error) error {
	return fmt.Errorf("%s: %w", action, err)
}
