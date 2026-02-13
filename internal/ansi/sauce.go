package ansi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// SAUCE record constants.
const (
	sauceIDString = "SAUCE"
	sauceRecSize  = 128
	sauceCommentIDString = "COMNT"
)

// SAUCE represents a Standard Architecture for Universal Comment Extensions record.
// This metadata format is appended to ANS, ASC, and other art files.
type SAUCE struct {
	Version   string
	Title     string
	Author    string
	Group     string
	Date      string
	FileSize  uint32
	DataType  byte
	FileType  byte
	TInfo1    uint16 // Width (for ANSI/ASCII)
	TInfo2    uint16 // Height (for ANSI/ASCII)
	TInfo3    uint16
	TInfo4    uint16
	Comments  byte
	Flags     byte
	TInfoS    string // SAUCE 00.5 font name
	CommentLines []string
}

// Width returns the display width from the SAUCE record, or 80 as default.
func (s *SAUCE) Width() int {
	if s.TInfo1 > 0 {
		return int(s.TInfo1)
	}
	return 80
}

// Height returns the display height from the SAUCE record, or 0 if unknown.
func (s *SAUCE) Height() int {
	return int(s.TInfo2)
}

// HasICEColors returns true if the file uses iCE colors (blink bit = bright background).
func (s *SAUCE) HasICEColors() bool {
	return s.Flags&0x01 != 0
}

// LetterSpacing returns the letter spacing mode from SAUCE flags.
// 0 = legacy, 1 = 8px, 2 = 9px
func (s *SAUCE) LetterSpacing() int {
	return int((s.Flags >> 1) & 0x03)
}

// AspectRatio returns the aspect ratio mode from SAUCE flags.
// 0 = legacy, 1 = stretch, 2 = square
func (s *SAUCE) AspectRatio() int {
	return int((s.Flags >> 3) & 0x03)
}

// ParseSAUCE extracts a SAUCE record from the end of file data.
// Returns the SAUCE record and the data without the SAUCE/comment block,
// or nil and the original data if no SAUCE record is found.
func ParseSAUCE(data []byte) (*SAUCE, []byte) {
	if len(data) < sauceRecSize {
		return nil, data
	}

	// SAUCE record is always at the last 128 bytes
	rec := data[len(data)-sauceRecSize:]

	// Check for SAUCE ID
	if string(rec[0:5]) != sauceIDString {
		return nil, data
	}

	s := &SAUCE{
		Version:  strings.TrimRight(string(rec[5:7]), "\x00 "),
		Title:    strings.TrimRight(string(rec[7:42]), "\x00 "),
		Author:   strings.TrimRight(string(rec[42:62]), "\x00 "),
		Group:    strings.TrimRight(string(rec[62:82]), "\x00 "),
		Date:     strings.TrimRight(string(rec[82:90]), "\x00 "),
		DataType: rec[94],
		FileType: rec[95],
		Comments: rec[104],
		Flags:    rec[105],
	}

	s.FileSize = binary.LittleEndian.Uint32(rec[90:94])
	s.TInfo1 = binary.LittleEndian.Uint16(rec[96:98])
	s.TInfo2 = binary.LittleEndian.Uint16(rec[98:100])
	s.TInfo3 = binary.LittleEndian.Uint16(rec[100:102])
	s.TInfo4 = binary.LittleEndian.Uint16(rec[102:104])

	// TInfoS (SAUCE 00.5 font name) at offset 106, 22 bytes
	if len(rec) >= 128 {
		s.TInfoS = strings.TrimRight(string(rec[106:128]), "\x00 ")
	}

	// Calculate the content end position
	contentEnd := len(data) - sauceRecSize

	// Read comment block if present
	if s.Comments > 0 {
		commentBlockSize := 5 + int(s.Comments)*64
		commentStart := contentEnd - commentBlockSize
		if commentStart >= 0 {
			commentBlock := data[commentStart:contentEnd]
			if string(commentBlock[0:5]) == sauceCommentIDString {
				s.CommentLines = make([]string, s.Comments)
				for i := 0; i < int(s.Comments); i++ {
					offset := 5 + i*64
					end := offset + 64
					if end > len(commentBlock) {
						break
					}
					s.CommentLines[i] = strings.TrimRight(string(commentBlock[offset:end]), "\x00 ")
				}
				contentEnd = commentStart
			}
		}
	}

	// Check for EOF character (0x1A) before SAUCE/comments
	if contentEnd > 0 && data[contentEnd-1] == 0x1A {
		contentEnd--
	}

	return s, data[:contentEnd]
}

// String returns a human-readable representation of the SAUCE record.
func (s *SAUCE) String() string {
	if s == nil {
		return "<no SAUCE>"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Title: %s\n", s.Title)
	fmt.Fprintf(&buf, "Author: %s\n", s.Author)
	fmt.Fprintf(&buf, "Group: %s\n", s.Group)
	fmt.Fprintf(&buf, "Date: %s\n", s.Date)
	fmt.Fprintf(&buf, "Size: %dx%d\n", s.Width(), s.Height())
	if s.TInfoS != "" {
		fmt.Fprintf(&buf, "Font: %s\n", s.TInfoS)
	}
	return buf.String()
}
