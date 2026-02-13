package ansi

import (
	"strconv"
	"strings"
)

// Field represents a placeholder found in a display file.
// Row/Col are 1-based terminal coordinates where the placeholder begins.
type Field struct {
	ID     string
	Row    int
	Col    int
	MaxLen int
}

// BlankPlaceholders returns a copy of data where any placeholder sequences
// of the form {{...}} are replaced with spaces (same length).
//
// This is used at render time so placeholders do not show on screen.
func BlankPlaceholders(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var out []byte // lazy copy
	i := 0
	for i < len(data) {
		if data[i] == '{' && i+1 < len(data) && data[i+1] == '{' {
			end := findPlaceholderEnd(data, i+2)
			if end != -1 {
				if out == nil {
					out = make([]byte, len(data))
					copy(out, data)
				}
				for j := i; j < end+2; j++ { // include '}}'
					out[j] = ' '
				}
				i = end + 2
				continue
			}
		}
		i++
	}

	if out == nil {
		return data
	}
	return out
}

// IndexFields scans a DisplayFile and returns placeholders of the form:
//
//	{{ID}} or {{ID,maxLen}}
//
// It does not modify the file content; it simulates cursor movement to compute
// the screen coordinates where placeholders appear.
func IndexFields(df *DisplayFile, termWidth int) map[string]Field {
	fields := make(map[string]Field)
	if df == nil || len(df.Data) == 0 {
		return fields
	}

	width := termWidth
	if width <= 0 {
		width = 80
	}

	row, col := 1, 1
	i := 0

	advancePrint := func(n int) {
		for n > 0 {
			if col > width {
				row++
				col = 1
			}
			col++
			n--
		}
	}

	for i < len(df.Data) {
		b := df.Data[i]

		// ANSI escape sequences
		if b == 0x1b { // ESC
			i++
			if i >= len(df.Data) {
				break
			}
			// CSI: ESC [
			if df.Data[i] == '[' {
				i++
				start := i
				for i < len(df.Data) {
					c := df.Data[i]
					if c >= 0x40 && c <= 0x7e { // final byte
						final := c
						params := string(df.Data[start:i])
						applyCSI(&row, &col, width, params, final)
						i++
						break
					}
					i++
				}
				continue
			}
			// Non-CSI escape: skip one byte (best-effort).
			i++
			continue
		}

		switch b {
		case '\r':
			col = 1
			i++
			continue
		case '\n':
			row++
			col = 1
			i++
			continue
		case '\b':
			if col > 1 {
				col--
			}
			i++
			continue
		}

		// Placeholder detection: {{...}}
		if b == '{' && i+1 < len(df.Data) && df.Data[i+1] == '{' {
			end := findPlaceholderEnd(df.Data, i+2)
			if end != -1 {
				payload := string(df.Data[i+2 : end])
				id, maxLen := parseFieldPayload(payload)
				if id != "" {
					if _, exists := fields[id]; !exists {
						fields[id] = Field{
							ID:     id,
							Row:    row,
							Col:    col,
							MaxLen: maxLen,
						}
					}
				}

				// Advance as if the placeholder text printed literally.
				advancePrint((end + 2) - i) // includes '{{' + payload + '}}'
				i = end + 2
				continue
			}
		}

		// Printable byte: advance cursor.
		if b >= 0x20 && b != 0x7f {
			advancePrint(1)
		}
		i++
	}

	return fields
}

func findPlaceholderEnd(data []byte, start int) int {
	// start points at first byte after '{{'
	for i := start; i+1 < len(data); i++ {
		if data[i] == '}' && data[i+1] == '}' {
			return i
		}
		// Do not attempt to parse through ESC sequences; placeholders are expected
		// to be literal printable bytes.
		if data[i] == 0x1b {
			return -1
		}
	}
	return -1
}

func parseFieldPayload(payload string) (id string, maxLen int) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", 0
	}
	parts := strings.SplitN(payload, ",", 2)
	id = strings.TrimSpace(parts[0])
	if id == "" {
		return "", 0
	}
	if len(parts) == 2 {
		nStr := strings.TrimSpace(parts[1])
		if n, err := strconv.Atoi(nStr); err == nil && n > 0 {
			maxLen = n
		}
	}
	return id, maxLen
}

func applyCSI(row, col *int, width int, params string, final byte) {
	// params is the substring between '[' and the final byte.
	// Examples:
	// - "10;20" with final 'H'
	// - "" with final 'm'
	// - "2" with final 'A'

	parseNums := func() []int {
		if params == "" {
			return nil
		}
		raw := strings.Split(params, ";")
		out := make([]int, 0, len(raw))
		for _, s := range raw {
			if s == "" {
				out = append(out, 0)
				continue
			}
			// Strip any non-digit prefixes (e.g. "?25h" we treat as unknown)
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil
			}
			out = append(out, n)
		}
		return out
	}

	nums := parseNums()

	switch final {
	case 'H', 'f': // CUP - Cursor Position
		r, c := 1, 1
		if len(nums) >= 1 && nums[0] > 0 {
			r = nums[0]
		}
		if len(nums) >= 2 && nums[1] > 0 {
			c = nums[1]
		}
		*row, *col = r, c
	case 'A': // CUU - Cursor Up
		n := 1
		if len(nums) >= 1 && nums[0] > 0 {
			n = nums[0]
		}
		*row -= n
		if *row < 1 {
			*row = 1
		}
	case 'B': // CUD - Cursor Down
		n := 1
		if len(nums) >= 1 && nums[0] > 0 {
			n = nums[0]
		}
		*row += n
	case 'C': // CUF - Cursor Forward
		n := 1
		if len(nums) >= 1 && nums[0] > 0 {
			n = nums[0]
		}
		*col += n
		if *col < 1 {
			*col = 1
		}
	case 'D': // CUB - Cursor Back
		n := 1
		if len(nums) >= 1 && nums[0] > 0 {
			n = nums[0]
		}
		*col -= n
		if *col < 1 {
			*col = 1
		}
	case 'J', 'K', 'm': // erase / SGR - no cursor move
		return
	default:
		// Unknown CSI - ignore.
	}

	if *col > width {
		*col = width
	}
	if *row < 1 {
		*row = 1
	}
	if *col < 1 {
		*col = 1
	}
}
