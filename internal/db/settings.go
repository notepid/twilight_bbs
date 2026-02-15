package db

import (
	"fmt"
)

// BBSSettings holds the BBS identity and limits from the database.
type BBSSettings struct {
	Name     string
	Sysop    string
	MaxNodes int
}

// GetBBSSettings retrieves the BBS settings from the database.
// Returns an error if the settings cannot be loaded.
func (db *DB) GetBBSSettings() (*BBSSettings, error) {
	var settings BBSSettings
	err := db.QueryRow("SELECT name, sysop, max_nodes FROM bbs_settings WHERE id = 1").Scan(
		&settings.Name,
		&settings.Sysop,
		&settings.MaxNodes,
	)
	if err != nil {
		return nil, fmt.Errorf("load bbs settings: %w", err)
	}
	return &settings, nil
}

// UpdateBBSSettings updates the BBS settings in the database.
func (db *DB) UpdateBBSSettings(settings *BBSSettings) error {
	_, err := db.Exec(
		"UPDATE bbs_settings SET name = ?, sysop = ?, max_nodes = ? WHERE id = 1",
		settings.Name,
		settings.Sysop,
		settings.MaxNodes,
	)
	if err != nil {
		return fmt.Errorf("update bbs settings: %w", err)
	}
	return nil
}
