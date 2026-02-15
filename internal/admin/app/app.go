package app

import (
	"fmt"
	"os"
	"time"

	"github.com/notepid/twilight_bbs/internal/config"
	"github.com/notepid/twilight_bbs/internal/db"
	"github.com/notepid/twilight_bbs/internal/filearea"
	"github.com/notepid/twilight_bbs/internal/message"
	"github.com/notepid/twilight_bbs/internal/user"
)

type App struct {
	ConfigPath string
	Config     *config.Config
	DBPath     string
	DB         *db.DB

	Users    *user.Repo
	Messages *message.Repo
	Files    *filearea.Repo

	BusyTimeout time.Duration
}

func New(configPath string) (*App, func(), error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, err
	}

	if err := os.MkdirAll(cfg.Paths.Data, 0755); err != nil {
		return nil, nil, fmt.Errorf("create data directory: %w", err)
	}

	database, err := db.Open(cfg.Paths.Database)
	if err != nil {
		return nil, nil, err
	}

	a := &App{
		ConfigPath:   configPath,
		Config:       cfg,
		DBPath:       cfg.Paths.Database,
		DB:           database,
		Users:        user.NewRepo(database.DB),
		Messages:     message.NewRepo(database.DB),
		Files:        filearea.NewRepo(database.DB),
		BusyTimeout:  5 * time.Second,
	}

	// Best-effort online use: reduce SQLITE_BUSY failures.
	_, _ = database.Exec("PRAGMA busy_timeout = 5000")

	cleanup := func() {
		_ = database.Close()
	}

	return a, cleanup, nil
}
