package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/notepid/twilight_bbs/internal/admin/app"
	"github.com/notepid/twilight_bbs/internal/db"
)

type settingsModel struct {
	app *app.App

	width  int
	height int

	Done bool

	form *huh.Form
	err  error

	name     string
	sysop    string
	maxNodes string
	save     bool
}

func newSettingsModel(a *app.App) *settingsModel {
	m := &settingsModel{app: a}

	settings, err := a.DB.GetBBSSettings()
	if err != nil {
		m.err = err
		return m
	}

	m.name = settings.Name
	m.sysop = settings.Sysop
	m.maxNodes = strconv.Itoa(settings.MaxNodes)

	m.form = buildSettingsForm(&m.name, &m.sysop, &m.maxNodes, &m.save)
	return m
}

func buildSettingsForm(name, sysop, maxNodes *string, save *bool) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("BBS Name").Value(name).Validate(nonEmpty("name")),
			huh.NewInput().Title("Sysop").Value(sysop).Validate(nonEmpty("sysop")),
			huh.NewInput().Title("Max Nodes").Value(maxNodes).Validate(validIntGreaterThan("max nodes", 0)),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Save changes?").Value(save),
		),
	)
}

func (m *settingsModel) SetSize(w, h int) {
	m.width, m.height = w, h
}

func (m *settingsModel) Update(msg tea.Msg) tea.Cmd {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.Done = true
			}
		}
		return nil
	}

	if m.form == nil {
		m.form = buildSettingsForm(&m.name, &m.sysop, &m.maxNodes, &m.save)
	}

	var cmd tea.Cmd
	updated, cmd := m.form.Update(msg)
	f, ok := updated.(*huh.Form)
	if !ok {
		m.err = fmt.Errorf("internal error: unexpected form model type")
		return nil
	}
	m.form = f

	if m.form.State == huh.StateCompleted {
		if m.save {
			maxNodesInt, _ := strconv.Atoi(strings.TrimSpace(m.maxNodes))
			settings := &db.BBSSettings{Name: strings.TrimSpace(m.name), Sysop: strings.TrimSpace(m.sysop), MaxNodes: maxNodesInt}
			if err := m.app.DB.UpdateBBSSettings(settings); err != nil {
				m.err = err
				return nil
			}
		}
		m.Done = true
		return nil
	}

	return cmd
}

func (m *settingsModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Settings error: %v\n\nPress Enter/Esc to go back.", m.err)
	}
	return m.form.View() + "\n\n(esc to go back)"
}

func nonEmpty(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s cannot be empty", field)
		}
		return nil
	}
}

func validIntGreaterThan(field string, min int) func(string) error {
	return func(s string) error {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("%s must be a number", field)
		}
		if v <= min {
			return fmt.Errorf("%s must be > %d", field, min)
		}
		return nil
	}
}
