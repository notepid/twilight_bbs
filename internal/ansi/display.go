package ansi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mikael/twilight_bbs/internal/terminal"
)

// DisplayFile represents a loaded ANSI or ASCII display file.
type DisplayFile struct {
	Name   string
	Path   string
	IsANSI bool
	Data   []byte
	Sauce  *SAUCE
}

// Loader handles finding and loading display files from a directory.
type Loader struct {
	baseDirs []string
}

// NewLoader creates a new display file loader that searches the given directories.
func NewLoader(dirs ...string) *Loader {
	return &Loader{baseDirs: dirs}
}

// Find locates a display file by name, preferring ANS over ASC.
// If ansiEnabled is false, it will only look for ASC files.
// The name should not include an extension.
func (l *Loader) Find(name string, ansiEnabled bool) (*DisplayFile, error) {
	safeName, err := sanitizeDisplayName(name)
	if err != nil {
		return nil, err
	}

	// Search order: prefer ANS when ANSI is enabled
	var extensions []string
	if ansiEnabled {
		extensions = []string{".ans", ".asc"}
	} else {
		extensions = []string{".asc", ".ans"} // ASC first, ANS as last resort
	}

	for _, dir := range l.baseDirs {
		for _, ext := range extensions {
			path := filepath.Join(dir, safeName+ext)
			if !isWithinBaseDir(dir, path) {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			isANSI := strings.EqualFold(ext, ".ans")

			// Parse SAUCE record if present
			sauce, content := ParseSAUCE(data)

			return &DisplayFile{
				Name:   safeName,
				Path:   path,
				IsANSI: isANSI,
				Data:   content,
				Sauce:  sauce,
			}, nil
		}
	}

	return nil, fmt.Errorf("display file not found: %s", safeName)
}

func sanitizeDisplayName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty display name")
	}
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("invalid display name")
	}

	clean := filepath.Clean(name)
	if clean == "." || clean == ".." {
		return "", fmt.Errorf("invalid display name")
	}
	if filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return "", fmt.Errorf("invalid display name")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid display name")
	}
	// Guard against Windows-style separators even on Unix.
	if strings.Contains(clean, "\\") {
		return "", fmt.Errorf("invalid display name")
	}

	return clean, nil
}

func isWithinBaseDir(base, path string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// Load reads a specific file by full path.
func (l *Loader) Load(path string) (*DisplayFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read display file %s: %w", path, err)
	}

	isANSI := strings.HasSuffix(strings.ToLower(path), ".ans")
	sauce, content := ParseSAUCE(data)

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	return &DisplayFile{
		Name:   name,
		Path:   path,
		IsANSI: isANSI,
		Data:   content,
		Sauce:  sauce,
	}, nil
}

// Display streams a display file to a terminal.
// For ANSI files, it sends the raw bytes (which contain ANSI escape sequences).
// For ASCII files, it sends with CRLF line endings.
func Display(term *terminal.Terminal, df *DisplayFile) error {
	if df.IsANSI && term.ANSIEnabled {
		return displayANSI(term, df)
	}
	return displayASCII(term, df)
}

// displayANSI streams an ANSI file to the terminal.
// ANSI files contain embedded escape sequences and use CP437 encoding.
func displayANSI(term *terminal.Terminal, df *DisplayFile) error {
	// Send the raw ANSI data - it already contains escape sequences
	// We send in chunks to allow for network buffering
	data := df.Data
	chunkSize := 1024

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		if err := term.SendBytes(data[i:end]); err != nil {
			return fmt.Errorf("display ANSI: %w", err)
		}

		// Small delay between chunks for the classic BBS "drawing" effect
		if end < len(data) {
			time.Sleep(5 * time.Millisecond)
		}
	}

	return nil
}

// displayASCII streams an ASCII file with CRLF line endings.
func displayASCII(term *terminal.Terminal, df *DisplayFile) error {
	lines := splitLines(df.Data)
	for _, line := range lines {
		if err := term.SendLn(string(line)); err != nil {
			return fmt.Errorf("display ASCII: %w", err)
		}
	}
	return nil
}

// splitLines splits data into lines, handling CR, LF, and CRLF.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[start:i]
			// Strip trailing CR
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	// Last line (may not end with newline)
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// DisplayWithPaging streams a display file with more-style paging.
func DisplayWithPaging(term *terminal.Terminal, df *DisplayFile, pageHeight int) error {
	if df.IsANSI && term.ANSIEnabled {
		// For ANSI files, we can't easily count lines due to escape sequences.
		// Just display without paging for now.
		return displayANSI(term, df)
	}

	// ASCII paging
	lines := splitLines(df.Data)
	lineCount := 0

	for _, line := range lines {
		if err := term.SendLn(string(line)); err != nil {
			return err
		}
		lineCount++

		if lineCount >= pageHeight-1 {
			if term.ANSIEnabled {
				term.Send(terminal.FgBrightCyan + " -- More -- " + terminal.Reset)
			} else {
				term.Send(" -- More -- ")
			}
			key, err := term.GetKey()
			if err != nil {
				return err
			}
			// Clear the "More" prompt
			term.Send("\r" + terminal.ClearLine())

			if key == 'q' || key == 'Q' || key == 27 { // q or ESC
				return nil
			}
			lineCount = 0
		}
	}

	return nil
}
