package transfer

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathValidator validates file paths are within authorized directories.
type PathValidator struct {
	allowedRoots []string
}

// NewPathValidator creates a path validator with authorized root directories.
func NewPathValidator(allowedRoots []string) *PathValidator {
	// Normalize all allowed roots to absolute paths
	normalized := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		abs, err := filepath.Abs(root)
		if err == nil {
			// Ensure trailing slash for consistent prefix matching
			if !strings.HasSuffix(abs, string(filepath.Separator)) {
				abs += string(filepath.Separator)
			}
			normalized = append(normalized, abs)
		}
	}
	return &PathValidator{allowedRoots: normalized}
}

// ValidatePath checks if a file path is within one of the allowed root directories.
// It prevents path traversal attacks by ensuring the resolved absolute path
// is within authorized boundaries.
func (v *PathValidator) ValidatePath(path string) error {
	if v == nil || len(v.allowedRoots) == 0 {
		// No validation configured - allow all paths (backward compatibility)
		return nil
	}

	// Resolve to absolute path and clean it (removes .., ., etc)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	// Ensure trailing separator for directory paths to prevent partial matches
	if !strings.HasSuffix(absPath, string(filepath.Separator)) {
		absPath += string(filepath.Separator)
	}

	// Check if path is within any allowed root
	for _, root := range v.allowedRoots {
		if strings.HasPrefix(absPath, root) {
			return nil
		}
		// Also check parent directory (for file paths)
		parentPath := filepath.Dir(filepath.Clean(path))
		if parentPath != "" {
			absParent, _ := filepath.Abs(parentPath)
			if absParent != "" && !strings.HasSuffix(absParent, string(filepath.Separator)) {
				absParent += string(filepath.Separator)
			}
			if strings.HasPrefix(absParent, root) {
				return nil
			}
		}
	}

	return fmt.Errorf("path is outside authorized directories: %s", absPath)
}

// ValidatePaths validates multiple paths.
func (v *PathValidator) ValidatePaths(paths []string) error {
	for _, path := range paths {
		if err := v.ValidatePath(path); err != nil {
			return err
		}
	}
	return nil
}
