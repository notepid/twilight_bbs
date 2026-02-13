package message

import "time"

// Area represents a message conference/area.
type Area struct {
	ID          int
	Name        string
	Description string
	ReadLevel   int
	WriteLevel  int
	SortOrder   int
	TotalMsgs   int // computed field
	NewMsgs     int // computed per-user
}

// Message represents a single message in an area.
type Message struct {
	ID         int
	AreaID     int
	FromUserID int
	FromName   string // joined from users table
	ToUserID   *int   // nil = public
	ToName     string // joined from users table
	Subject    string
	Body       string
	ReplyToID  *int
	CreatedAt  time.Time
}
