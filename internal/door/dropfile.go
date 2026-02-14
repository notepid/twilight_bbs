package door

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteDoorSys generates a DOOR.SYS drop file.
// DOOR.SYS is the most widely supported drop file format.
func WriteDoorSys(dir string, s *Session) (string, error) {
	path := filepath.Join(dir, "DOOR.SYS")

	u := s.User
	now := time.Now()

	// DOOR.SYS format - 52 lines
	lines := []string{
		fmt.Sprintf("COM%d:", s.ComPort),          // 1: COM port
		fmt.Sprintf("%d", s.BaudRate),              // 2: baud rate
		"8",                                         // 3: data bits
		fmt.Sprintf("%d", s.NodeID),                // 4: node number
		fmt.Sprintf("%d", s.BaudRate),              // 5: DTE rate
		"Y",                                         // 6: screen display
		"Y",                                         // 7: printer toggle
		"Y",                                         // 8: page bell
		"Y",                                         // 9: caller alarm
		u.Username,                                  // 10: user name
		u.Location,                                  // 11: calling from
		"",                                          // 12: home phone
		"",                                          // 13: work phone
		"",                                          // 14: password (never sent)
		fmt.Sprintf("%d", u.SecurityLevel),         // 15: security level
		fmt.Sprintf("%d", u.TotalCalls),            // 16: total calls
		now.Format("01/02/2006"),                    // 17: last call date
		fmt.Sprintf("%d", s.TimeLeftMins*60),       // 18: seconds remaining
		fmt.Sprintf("%d", s.TimeLeftMins),          // 19: minutes remaining
		"GR",                                        // 20: graphics mode (GR=ANSI)
		"25",                                        // 21: screen height
		"Y",                                         // 22: expert mode
		"",                                          // 23: conferences registered
		"",                                          // 24: current conference
		"",                                          // 25: expiration date
		fmt.Sprintf("%d", u.ID),                    // 26: user record number
		"Y",                                         // 27: default protocol
		"0",                                         // 28: total uploads
		"0",                                         // 29: total downloads
		"0",                                         // 30: daily download K
		"999999",                                    // 31: daily download K limit
		now.Format("01/02/2006"),                    // 32: caller's birthday
		"",                                          // 33: path to user files
		"",                                          // 34: path to door files
		now.Format("15:04"),                         // 35: time of this call
		now.Format("15:04"),                         // 36: time of last call
		"32768",                                     // 37: max daily files
		"0",                                         // 38: files downloaded today
		"0",                                         // 39: total uploaded K
		"0",                                         // 40: total downloaded K
		"",                                          // 41: user comment
		"0",                                         // 42: doors opened
		"0",                                         // 43: msgs left
	}

	content := strings.Join(lines, "\r\n") + "\r\n"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create drop file dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write DOOR.SYS: %w", err)
	}

	return path, nil
}

// WriteDorInfo generates a DORINFO1.DEF drop file.
func WriteDorInfo(dir string, s *Session) (string, error) {
	filename := fmt.Sprintf("DORINFO%d.DEF", s.NodeID)
	path := filepath.Join(dir, filename)

	u := s.User
	parts := strings.SplitN(u.RealName, " ", 2)
	firstName := u.Username
	lastName := ""
	if len(parts) >= 1 && parts[0] != "" {
		firstName = parts[0]
	}
	if len(parts) >= 2 {
		lastName = parts[1]
	}

	lines := []string{
		"Twilight BBS",                         // 1: BBS name
		"Sysop",                                // 2: sysop first name
		"",                                     // 3: sysop last name
		fmt.Sprintf("COM%d", s.ComPort),       // 4: COM port
		fmt.Sprintf("%d BAUD,N,8,1", s.BaudRate), // 5: baud rate
		"0",                                    // 6: network type
		firstName,                              // 7: user first name
		lastName,                               // 8: user last name
		u.Location,                             // 9: user location
		"1",                                    // 10: ANSI mode (0=no, 1=yes)
		fmt.Sprintf("%d", u.SecurityLevel),    // 11: security level
		fmt.Sprintf("%d", s.TimeLeftMins),     // 12: minutes remaining
		"-1",                                   // 13: fossil flag (-1=door handles)
	}

	content := strings.Join(lines, "\r\n") + "\r\n"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create drop file dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", filename, err)
	}

	return path, nil
}

// WriteDropFile writes the appropriate drop file based on the session config.
func WriteDropFile(dir string, s *Session) (string, error) {
	switch s.DoorConfig.DropFileType {
	case "DORINFO1.DEF":
		return WriteDorInfo(dir, s)
	default:
		return WriteDoorSys(dir, s)
	}
}
