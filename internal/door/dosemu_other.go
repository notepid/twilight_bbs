//go:build !linux

package door

import (
	"fmt"
	"io"
)

// Launch is not supported on non-Linux platforms.
func (l *Launcher) Launch(session *Session, stdin io.Reader, stdout io.Writer) error {
	return fmt.Errorf("DOS doors require Linux (dosemu2 is not available on this platform)")
}
