//go:build linux

package transfer

import (
	"net"
	"os"
	"syscall"
)

// createSocketPair creates a Unix socketpair and returns the parent and child
// ends as a net.Conn and an *os.File respectively. The caller should:
//   - Use parentConn for reading/writing to the child process
//   - Pass childFile as stdin+stdout of the child process
//   - Close childFile after the child process inherits it
//   - Close parentConn after the transfer is complete
func createSocketPair() (parentConn net.Conn, childFile *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}

	parentFile := os.NewFile(uintptr(fds[0]), "sexyz-parent")
	childFile = os.NewFile(uintptr(fds[1]), "sexyz-child")

	parentConn, err = net.FileConn(parentFile)
	parentFile.Close() // FileConn dups the fd
	if err != nil {
		childFile.Close()
		return nil, nil, err
	}

	return parentConn, childFile, nil
}
