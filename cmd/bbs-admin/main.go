package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/notepid/twilight_bbs/internal/admin/app"
	"github.com/notepid/twilight_bbs/internal/admin/ui"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	a, cleanup, err := app.New(*configPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer cleanup()

	p := tea.NewProgram(ui.NewRootModel(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
