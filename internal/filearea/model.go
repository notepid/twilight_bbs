package filearea

import "time"

// Area represents a file download/upload area.
type Area struct {
	ID            int
	Name          string
	Description   string
	DiskPath      string
	DownloadLevel int
	UploadLevel   int
	SortOrder     int
	FileCount     int // computed field
}

// Entry represents a file in a file area.
type Entry struct {
	ID            int
	AreaID        int
	Filename      string
	Description   string
	SizeBytes     int64
	UploaderID    int
	UploaderName  string // joined
	DownloadCount int
	UploadedAt    time.Time
}
