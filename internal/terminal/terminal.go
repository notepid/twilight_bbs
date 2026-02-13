package terminal

import (
	"fmt"
	"io"
	"strings"
)

// Terminal provides a high-level read/write abstraction over a raw
// connection. It handles CRLF line endings and provides BBS-oriented
// I/O methods.
type Terminal struct {
	rwc         io.ReadWriteCloser
	Width       int
	Height      int
	ANSIEnabled bool

	// echoControl is called to enable/disable safe echo behavior.
	// For telnet, this typically controls whether the client performs local echo.
	echoControl func(on bool) error
}

// New creates a new Terminal wrapping the given ReadWriteCloser.
func New(rwc io.ReadWriteCloser, width, height int, ansiEnabled bool) *Terminal {
	return &Terminal{
		rwc:         rwc,
		Width:       width,
		Height:      height,
		ANSIEnabled: ansiEnabled,
	}
}

// SetEchoControl registers a callback for enabling/disabling echo behavior.
func (t *Terminal) SetEchoControl(fn func(on bool) error) {
	t.echoControl = fn
}

// Close closes the underlying connection.
func (t *Terminal) Close() error {
	return t.rwc.Close()
}

// Read implements io.Reader, delegating to the underlying connection.
func (t *Terminal) Read(p []byte) (int, error) {
	return t.rwc.Read(p)
}

// Write implements io.Writer, delegating to the underlying connection.
func (t *Terminal) Write(p []byte) (int, error) {
	return t.rwc.Write(p)
}

// Send writes raw bytes to the terminal.
func (t *Terminal) Send(data string) error {
	_, err := io.WriteString(t.rwc, data)
	return err
}

// SendBytes writes raw bytes to the terminal.
func (t *Terminal) SendBytes(data []byte) error {
	_, err := t.rwc.Write(data)
	return err
}

// SendLn writes a line of text followed by CR+LF.
func (t *Terminal) SendLn(text string) error {
	return t.Send(text + "\r\n")
}

// Cls clears the screen.
func (t *Terminal) Cls() error {
	if t.ANSIEnabled {
		return t.Send(ClearScreen())
	}
	// ASCII fallback: send 24 blank lines
	return t.Send(strings.Repeat("\r\n", 24))
}

// GotoXY positions the cursor (1-based row, col).
func (t *Terminal) GotoXY(row, col int) error {
	if t.ANSIEnabled {
		return t.Send(MoveTo(row, col))
	}
	return nil // no-op for ASCII
}

// SetColor sets the text color using ANSI SGR codes.
func (t *Terminal) SetColor(fg, bg int) error {
	if t.ANSIEnabled {
		return t.Send(Color(fg, bg))
	}
	return nil
}

// ResetColor resets to default colors.
func (t *Terminal) ResetColor() error {
	if t.ANSIEnabled {
		return t.Send(Reset)
	}
	return nil
}

// ReadByte reads a single byte from the terminal.
func (t *Terminal) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := t.rwc.Read(buf)
	return buf[0], err
}

// GetKey waits for and returns a single keypress.
func (t *Terminal) GetKey() (byte, error) {
	return t.ReadByte()
}

// GetLine reads a line of input up to maxLen characters, with echo.
// Returns the entered string (without trailing CR/LF).
func (t *Terminal) GetLine(maxLen int) (string, error) {
	var buf []byte
	for {
		b, err := t.ReadByte()
		if err != nil {
			return string(buf), err
		}

		switch b {
		case '\r', '\n':
			t.Send("\r\n")
			return string(buf), nil
		case 8, 127: // backspace or delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				t.Send("\b \b")
			}
		default:
			if b >= 32 && b < 127 && len(buf) < maxLen {
				buf = append(buf, b)
				t.Send(string(b))
			}
		}
	}
}

// GetPassword reads a line of input without echo, displaying asterisks.
func (t *Terminal) GetPassword(maxLen int) (string, error) {
	// Disable echo
	if t.echoControl != nil {
		t.echoControl(false)
	}

	var buf []byte
	for {
		b, err := t.ReadByte()
		if err != nil {
			// Re-enable echo before returning
			if t.echoControl != nil {
				t.echoControl(true)
			}
			return string(buf), err
		}

		switch b {
		case '\r', '\n':
			// Re-enable echo
			if t.echoControl != nil {
				t.echoControl(true)
			}
			t.Send("\r\n")
			return string(buf), nil
		case 8, 127:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				t.Send("\b \b")
			}
		default:
			if b >= 32 && b < 127 && len(buf) < maxLen {
				buf = append(buf, b)
				t.Send("*")
			}
		}
	}
}

// Pause displays "Press any key to continue..." and waits for a keypress.
func (t *Terminal) Pause() error {
	if t.ANSIEnabled {
		t.Send(FgBrightCyan + "Press any key to continue..." + Reset)
	} else {
		t.Send("Press any key to continue...")
	}
	_, err := t.GetKey()
	t.Send("\r\n")
	return err
}

// YesNo displays a prompt and waits for Y or N.
func (t *Terminal) YesNo(prompt string) (bool, error) {
	t.Send(fmt.Sprintf("%s (Y/N) ", prompt))
	for {
		b, err := t.GetKey()
		if err != nil {
			return false, err
		}
		switch b {
		case 'Y', 'y':
			t.SendLn("Yes")
			return true, nil
		case 'N', 'n':
			t.SendLn("No")
			return false, nil
		}
	}
}

// Hotkey displays a prompt and waits for a single keypress, returning it.
func (t *Terminal) Hotkey(prompt string) (byte, error) {
	t.Send(prompt)
	return t.GetKey()
}

// Ask displays a prompt and reads a line of input.
func (t *Terminal) Ask(prompt string, maxLen int) (string, error) {
	t.Send(prompt)
	return t.GetLine(maxLen)
}
