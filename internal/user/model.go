package user

import "time"

// User represents a BBS user account.
type User struct {
	ID            int
	Username      string
	PasswordHash  string
	RealName      string
	Location      string
	Email         string
	SecurityLevel int
	TotalCalls    int
	LastCallAt    *time.Time
	ANSIEnabled   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SecurityLevel constants following classic BBS conventions.
const (
	LevelNew      = 10  // New user (just registered)
	LevelValidated = 20 // Validated user
	LevelRegular  = 30  // Regular user
	LevelTrusted  = 50  // Trusted user
	LevelCoSysop  = 90  // Co-sysop
	LevelSysop    = 100 // Sysop (full access)
)
