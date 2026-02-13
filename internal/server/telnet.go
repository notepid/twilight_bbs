package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

// Telnet protocol constants.
const (
	IAC  byte = 255 // Interpret As Command
	DONT byte = 254
	DO   byte = 253
	WONT byte = 252
	WILL byte = 251
	SB   byte = 250 // Sub-negotiation Begin
	SE   byte = 240 // Sub-negotiation End
	GA   byte = 249 // Go Ahead

	// Telnet options
	OptEcho    byte = 1  // Echo
	OptSGA     byte = 3  // Suppress Go Ahead
	OptTType   byte = 24 // Terminal Type
	OptNAWS    byte = 31 // Negotiate About Window Size
	OptLinemod byte = 34 // Linemode
)

// TelnetConn wraps a raw TCP connection with telnet protocol handling.
// It transparently strips IAC sequences from the data stream and
// provides methods for telnet negotiation.
type TelnetConn struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex

	// Terminal properties discovered via negotiation
	TermType    string
	Width       int
	Height      int
	ANSICapable bool
}

// NewTelnetConn wraps a raw TCP connection with telnet protocol handling.
func NewTelnetConn(conn net.Conn) *TelnetConn {
	return &TelnetConn{
		conn:        conn,
		reader:      bufio.NewReaderSize(conn, 1024),
		Width:       80,
		Height:      24,
		ANSICapable: true, // assume ANSI until told otherwise
	}
}

// Negotiate sends initial telnet option negotiations.
func (tc *TelnetConn) Negotiate() error {
	// WILL ECHO - we control echo (important for password prompts)
	if err := tc.sendCommand(WILL, OptEcho); err != nil {
		return err
	}
	// WILL SGA - suppress go-ahead for character-at-a-time mode
	if err := tc.sendCommand(WILL, OptSGA); err != nil {
		return err
	}
	// DO SGA - request the client suppress go-ahead as well
	if err := tc.sendCommand(DO, OptSGA); err != nil {
		return err
	}
	// DONT LINEMODE - request character-at-a-time input (avoids local line editing/echo)
	if err := tc.sendCommand(DONT, OptLinemod); err != nil {
		return err
	}
	// DO NAWS - request window size from client
	if err := tc.sendCommand(DO, OptNAWS); err != nil {
		return err
	}
	// DO TTYPE - request terminal type from client
	if err := tc.sendCommand(DO, OptTType); err != nil {
		return err
	}
	return nil
}

// sendCommand sends a 3-byte IAC command sequence.
func (tc *TelnetConn) sendCommand(cmd, option byte) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	_, err := tc.conn.Write([]byte{IAC, cmd, option})
	return err
}

// ReadByte reads a single byte from the connection, handling IAC sequences.
func (tc *TelnetConn) ReadByte() (byte, error) {
	for {
		b, err := tc.reader.ReadByte()
		if err != nil {
			return 0, err
		}

		if b != IAC {
			return b, nil
		}

		// IAC received - read the command byte
		cmd, err := tc.reader.ReadByte()
		if err != nil {
			return 0, err
		}

		switch cmd {
		case IAC:
			// Escaped IAC (255) - return literal 0xFF
			return IAC, nil

		case WILL, WONT:
			// Client is offering/refusing an option
			opt, err := tc.reader.ReadByte()
			if err != nil {
				return 0, err
			}
			tc.handleWillWont(cmd, opt)
			continue

		case DO, DONT:
			// Client is requesting/refusing an option from us
			opt, err := tc.reader.ReadByte()
			if err != nil {
				return 0, err
			}
			tc.handleDoDont(cmd, opt)
			continue

		case SB:
			// Sub-negotiation
			if err := tc.handleSubNegotiation(); err != nil {
				return 0, err
			}
			continue

		case GA:
			// Go Ahead - ignore
			continue

		default:
			// Unknown command - skip
			continue
		}
	}
}

// Read implements io.Reader, filtering telnet protocol from the data stream.
func (tc *TelnetConn) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		b, err := tc.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		p[n] = b
		n++

		// Don't block waiting for more data if buffer has content
		if n > 0 && tc.reader.Buffered() == 0 {
			break
		}
	}
	return n, nil
}

// Write sends data to the client, escaping any literal 0xFF bytes.
func (tc *TelnetConn) Write(p []byte) (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	written := 0
	for i, b := range p {
		if b == IAC {
			// Write everything up to this byte
			if i > written {
				if _, err := tc.conn.Write(p[written:i]); err != nil {
					return written, err
				}
			}
			// Escape the IAC byte
			if _, err := tc.conn.Write([]byte{IAC, IAC}); err != nil {
				return i, err
			}
			written = i + 1
		}
	}
	// Write remaining bytes
	if written < len(p) {
		if _, err := tc.conn.Write(p[written:]); err != nil {
			return written, err
		}
	}
	return len(p), nil
}

// WriteRaw writes data directly without IAC escaping (for sending telnet commands).
func (tc *TelnetConn) WriteRaw(p []byte) (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.conn.Write(p)
}

// Close closes the underlying connection.
func (tc *TelnetConn) Close() error {
	return tc.conn.Close()
}

// RemoteAddr returns the remote address of the connection.
func (tc *TelnetConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

// SetEcho enables or disables server-side echo.
func (tc *TelnetConn) SetEcho(on bool) error {
	// Telnet ECHO negotiation controls whether the client should perform local echo.
	// If we send WONT ECHO, many clients switch to local echo, which would leak
	// password characters while the server is printing '*'.
	//
	// This callback is used by terminal.GetPassword to hide plaintext input, so we
	// always keep the server in "echo mode" from the client's perspective.
	_ = on
	return tc.sendCommand(WILL, OptEcho)
}

// handleWillWont processes WILL/WONT responses from the client.
func (tc *TelnetConn) handleWillWont(cmd, opt byte) {
	switch opt {
	case OptNAWS:
		// Client will send window size - good, we'll get it in SB
	case OptTType:
		if cmd == WILL {
			// Client supports terminal type - request it
			// Send SB TTYPE SEND SE
			tc.mu.Lock()
			tc.conn.Write([]byte{IAC, SB, OptTType, 1, IAC, SE})
			tc.mu.Unlock()
		}
	case OptLinemod:
		// Client offered linemode; refuse so we get character-at-a-time input.
		if cmd == WILL {
			_ = tc.sendCommand(DONT, OptLinemod)
		}
	}
}

// handleDoDont processes DO/DONT requests from the client.
func (tc *TelnetConn) handleDoDont(cmd, opt byte) {
	switch opt {
	case OptEcho, OptSGA:
		// We already said WILL for these, client confirms with DO
	default:
		if cmd == DO {
			// We don't support this option - send WONT
			tc.sendCommand(WONT, opt)
		}
	}
}

// handleSubNegotiation reads and processes a sub-negotiation sequence.
func (tc *TelnetConn) handleSubNegotiation() error {
	// Read until IAC SE
	const maxSubnegLen = 1024
	var buf []byte
	for {
		b, err := tc.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("subneg read: %w", err)
		}
		if b == IAC {
			next, err := tc.reader.ReadByte()
			if err != nil {
				return fmt.Errorf("subneg read: %w", err)
			}
			if next == SE {
				break
			}
			if next == IAC {
				buf = append(buf, IAC)
				if len(buf) > maxSubnegLen {
					return fmt.Errorf("subneg too long")
				}
				continue
			}
			// Unexpected - treat as end
			break
		}
		buf = append(buf, b)
		if len(buf) > maxSubnegLen {
			return fmt.Errorf("subneg too long")
		}
	}

	if len(buf) == 0 {
		return nil
	}

	switch buf[0] {
	case OptNAWS:
		// NAWS: option(1) + width(2) + height(2) = 5 bytes
		if len(buf) >= 5 {
			tc.Width = int(buf[1])<<8 | int(buf[2])
			tc.Height = int(buf[3])<<8 | int(buf[4])
		}
	case OptTType:
		// TTYPE: option(1) + IS(1) + type string
		if len(buf) >= 2 && buf[1] == 0 {
			term := string(buf[2:])
			if len(term) > 64 {
				term = term[:64]
			}
			tc.TermType = term
			tc.ANSICapable = isANSITermType(tc.TermType)
		}
	}

	return nil
}

// isANSITermType checks if a terminal type string indicates ANSI support.
func isANSITermType(termType string) bool {
	switch termType {
	case "ANSI", "ansi", "ANSI-BBS", "ansi-bbs",
		"xterm", "xterm-256color", "xterm-color",
		"vt100", "VT100", "vt102", "VT102",
		"linux", "screen", "screen-256color",
		"tmux", "tmux-256color",
		"rxvt", "rxvt-unicode":
		return true
	}
	return false
}

// Ensure TelnetConn implements io.ReadWriteCloser.
var _ io.ReadWriteCloser = (*TelnetConn)(nil)
