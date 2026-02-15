package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/notepid/twilight_bbs/internal/admin/app"
)

type screen int

const (
	screenHome screen = iota
	screenSettings
	screenUsers
	screenMessages
	screenFiles
)

type rootModel struct {
	app *app.App

	width  int
	height int

	active screen

	homeList list.Model
	err     error

	settings *settingsModel
	users    *usersModel
	messages *messagesModel
	files    *filesModel
}

type menuItem struct {
	title string
	desc  string
	to    screen
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.desc }
func (m menuItem) FilterValue() string { return m.title }

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
)

func NewRootModel(a *app.App) tea.Model {
	items := []list.Item{
		menuItem{title: "BBS Settings", desc: "Edit BBS name, sysop, max nodes", to: screenSettings},
		menuItem{title: "Users", desc: "Manage user accounts", to: screenUsers},
		menuItem{title: "Messages", desc: "View message areas and messages", to: screenMessages},
		menuItem{title: "File Areas", desc: "View file areas and entries", to: screenFiles},
		menuItem{title: "Quit", desc: "Exit", to: -1},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Twilight BBS Admin"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)

	return &rootModel{
		app:      a,
		active:   screenHome,
		homeList: l,
	}
}

func (m *rootModel) Init() tea.Cmd {
	return nil
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.homeList.SetSize(msg.Width, msg.Height-2)
		if m.settings != nil {
			m.settings.SetSize(msg.Width, msg.Height)
		}
		if m.users != nil {
			m.users.SetSize(msg.Width, msg.Height)
		}
		if m.messages != nil {
			m.messages.SetSize(msg.Width, msg.Height)
		}
		if m.files != nil {
			m.files.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	switch m.active {
	case screenHome:
		return m.updateHome(msg)
	case screenSettings:
		if m.settings == nil {
			m.settings = newSettingsModel(m.app)
			m.settings.SetSize(m.width, m.height)
		}
		cmd := m.settings.Update(msg)
		if m.settings.Done {
			m.active = screenHome
			m.settings = nil
		}
		return m, cmd
	case screenUsers:
		if m.users == nil {
			m.users = newUsersModel(m.app)
			m.users.SetSize(m.width, m.height)
		}
		cmd := m.users.Update(msg)
		if m.users.Done {
			m.active = screenHome
			m.users = nil
		}
		return m, cmd
	case screenMessages:
		if m.messages == nil {
			m.messages = newMessagesModel(m.app)
			m.messages.SetSize(m.width, m.height)
		}
		cmd := m.messages.Update(msg)
		if m.messages.Done {
			m.active = screenHome
			m.messages = nil
		}
		return m, cmd
	case screenFiles:
		if m.files == nil {
			m.files = newFilesModel(m.app)
			m.files.SetSize(m.width, m.height)
		}
		cmd := m.files.Update(msg)
		if m.files.Done {
			m.active = screenHome
			m.files = nil
		}
		return m, cmd
	default:
		return m, nil
	}
}

func (m *rootModel) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.homeList, cmd = m.homeList.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if it, ok := m.homeList.SelectedItem().(menuItem); ok {
				if it.to == -1 {
					return m, tea.Quit
				}
				m.activate(it.to)
				return m, nil
			}
		}
	}

	return m, cmd
}

func (m *rootModel) activate(s screen) {
	m.active = s

	switch s {
	case screenSettings:
		if m.settings == nil {
			m.settings = newSettingsModel(m.app)
			m.settings.SetSize(m.width, m.height)
		}
	case screenUsers:
		if m.users == nil {
			m.users = newUsersModel(m.app)
			m.users.SetSize(m.width, m.height)
		}
	case screenMessages:
		if m.messages == nil {
			m.messages = newMessagesModel(m.app)
			m.messages.SetSize(m.width, m.height)
		}
	case screenFiles:
		if m.files == nil {
			m.files = newFilesModel(m.app)
			m.files.SetSize(m.width, m.height)
		}
	}
}

func (m *rootModel) View() string {
	if m.err != nil {
		return errStyle.Render("Error: ") + m.err.Error()
	}

	switch m.active {
	case screenHome:
		return m.homeList.View()
	case screenSettings:
		if m.settings == nil {
			return "Loading settings..."
		}
		return m.settings.View()
	case screenUsers:
		if m.users == nil {
			return "Loading users..."
		}
		return m.users.View()
	case screenMessages:
		if m.messages == nil {
			return "Loading messages..."
		}
		return m.messages.View()
	case screenFiles:
		if m.files == nil {
			return "Loading files..."
		}
		return m.files.View()
	default:
		return titleStyle.Render("Unknown screen") + "\n" + fmt.Sprint(m.active)
	}
}
