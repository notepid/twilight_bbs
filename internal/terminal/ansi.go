package terminal

import "fmt"

// ANSI color constants matching the BBS standard.
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Underscore = "\033[4m"
	Blink      = "\033[5m"
	Reverse    = "\033[7m"
	Hidden     = "\033[8m"

	// Foreground colors
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgBrown   = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgGray    = "\033[37m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgBrown   = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgGray    = "\033[47m"
)

// Bright foreground color helpers (Bold + foreground).
const (
	FgDarkGray      = "\033[1;30m"
	FgBrightRed     = "\033[1;31m"
	FgBrightGreen   = "\033[1;32m"
	FgYellow        = "\033[1;33m"
	FgBrightBlue    = "\033[1;34m"
	FgBrightMagenta = "\033[1;35m"
	FgBrightCyan    = "\033[1;36m"
	FgWhite         = "\033[1;37m"
)

// ClearScreen sends the ANSI clear-screen sequence and homes the cursor.
func ClearScreen() string {
	return "\033[2J\033[1;1H"
}

// MoveTo returns an ANSI cursor positioning sequence.
func MoveTo(row, col int) string {
	return fmt.Sprintf("\033[%d;%dH", row, col)
}

// CursorUp returns an ANSI cursor-up sequence.
func CursorUp(n int) string {
	if n <= 0 {
		n = 1
	}
	return fmt.Sprintf("\033[%dA", n)
}

// CursorDown returns an ANSI cursor-down sequence.
func CursorDown(n int) string {
	if n <= 0 {
		n = 1
	}
	return fmt.Sprintf("\033[%dB", n)
}

// CursorRight returns an ANSI cursor-right sequence.
func CursorRight(n int) string {
	if n <= 0 {
		n = 1
	}
	return fmt.Sprintf("\033[%dC", n)
}

// CursorLeft returns an ANSI cursor-left sequence.
func CursorLeft(n int) string {
	if n <= 0 {
		n = 1
	}
	return fmt.Sprintf("\033[%dD", n)
}

// Color returns an ANSI SGR sequence for the given foreground and background.
// fg: 30-37, bg: 40-47. Pass -1 to leave unchanged.
func Color(fg, bg int) string {
	if fg >= 0 && bg >= 0 {
		return fmt.Sprintf("\033[%d;%dm", fg, bg)
	}
	if fg >= 0 {
		return fmt.Sprintf("\033[%dm", fg)
	}
	if bg >= 0 {
		return fmt.Sprintf("\033[%dm", bg)
	}
	return ""
}

// SaveCursor returns the ANSI save-cursor-position sequence.
func SaveCursor() string {
	return "\033[s"
}

// RestoreCursor returns the ANSI restore-cursor-position sequence.
func RestoreCursor() string {
	return "\033[u"
}

// HideCursor returns the ANSI hide-cursor sequence.
func HideCursor() string {
	return "\033[?25l"
}

// ShowCursor returns the ANSI show-cursor sequence.
func ShowCursor() string {
	return "\033[?25h"
}

// ClearLine clears the current line.
func ClearLine() string {
	return "\033[2K"
}

// ClearToEOL clears from cursor to end of line.
func ClearToEOL() string {
	return "\033[0K"
}

// ResetTerminal sends the full terminal reset sequence (ESC c).
func ResetTerminal() string {
	return "\033c"
}
