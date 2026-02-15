//go:build !linux

package transfer

import (
	"fmt"
	"net"
	"os"
)

// createSocketPair is not supported on non-Linux platforms.
func createSocketPair() (net.Conn, *os.File, error) {
	return nil, nil, fmt.Errorf("socketpair not supported on this platform")
}
