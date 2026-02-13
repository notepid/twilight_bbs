package door

import "github.com/notepid/twilight_bbs/internal/user"

// Config holds configuration for a single door.
type Config struct {
	ID            int
	Name          string
	Description   string
	Command       string // DOS executable path (relative to drive C)
	DropFileType  string // "DOOR.SYS" or "DORINFO1.DEF"
	SecurityLevel int
}

// Session holds the context for a door session.
type Session struct {
	DoorConfig    *Config
	User          *user.User
	NodeID        int
	TimeLeftMins  int
	ComPort       int // emulated COM port (usually 0 for local)
	BaudRate      int
	DropFilePath  string
	DosemuPath    string
	DriveCPath    string
}
