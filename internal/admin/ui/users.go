package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/huh"

	"github.com/notepid/twilight_bbs/internal/admin/app"
	"github.com/notepid/twilight_bbs/internal/user"
)

type usersModel struct {
	app *app.App

	width  int
	height int

	Done bool

	state usersState

	list list.Model
	err  error

	selected *user.User

	form *huh.Form

	createUsername string
	createPassword string
	createRealName string
	createLocation string
	createEmail    string
	createSave     bool

	editRealName string
	editLocation string
	editEmail    string
	editSave     bool

	newPassword string
	pwConfirm   string
	pwSave      bool

	levelChoice string
	levelSave   bool

	ansiEnabled bool
	ansiSave    bool
}

type usersState int

const (
	usersStateList usersState = iota
	usersStateDetail
	usersStateCreate
	usersStateEditProfile
	usersStateResetPassword
	usersStateSetLevel
	usersStateSetANSI
)

type userItem struct {
	id    int
	title string
	desc  string
	kind  string
}

func (i userItem) Title() string       { return i.title }
func (i userItem) Description() string { return i.desc }
func (i userItem) FilterValue() string { return i.title }

func newUsersModel(a *app.App) *usersModel {
	m := &usersModel{app: a, state: usersStateList}
	m.reloadList()
	return m
}

func (m *usersModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w, h-2)
}

func (m *usersModel) Update(msg tea.Msg) tea.Cmd {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.err = nil
				m.state = usersStateList
				m.form = nil
				m.selected = nil
				m.reloadList()
			}
		}
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.state == usersStateList {
				m.Done = true
				return nil
			}
		case "esc":
			m.back()
			return nil
		}
	}

	switch m.state {
	case usersStateList:
		return m.updateList(msg)
	case usersStateDetail:
		return m.updateDetail(msg)
	case usersStateCreate, usersStateEditProfile, usersStateResetPassword, usersStateSetLevel, usersStateSetANSI:
		return m.updateForm(msg)
	default:
		return nil
	}
}

func (m *usersModel) updateList(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			it, ok := m.list.SelectedItem().(userItem)
			if !ok {
				return cmd
			}
			if it.kind == "create" {
				m.startCreate()
				return nil
			}

			u, err := m.app.Users.GetByID(it.id)
			if err != nil {
				m.err = err
				return nil
			}
			m.selected = u
			m.state = usersStateDetail
			m.list = newActionList(m.width, m.height)
			return nil
		}
	}

	return cmd
}

func (m *usersModel) updateDetail(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			it, ok := m.list.SelectedItem().(userItem)
			if !ok {
				return cmd
			}
			switch it.kind {
			case "edit_profile":
				m.startEditProfile()
			case "set_level":
				m.startSetLevel()
			case "set_ansi":
				m.startSetANSI()
			case "reset_password":
				m.startResetPassword()
			case "back":
				m.back()
			}
			return nil
		}
	}

	return cmd
}

func (m *usersModel) updateForm(msg tea.Msg) tea.Cmd {
	if m.form == nil {
		m.err = fmt.Errorf("internal error: form not initialized")
		return nil
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
		switch m.state {
		case usersStateCreate:
			if m.createSave {
				if m.app.Users.Exists(m.createUsername) {
					m.err = fmt.Errorf("username already exists")
					return nil
				}
				_, err := m.app.Users.Create(m.createUsername, m.createPassword, m.createRealName, m.createLocation, m.createEmail)
				if err != nil {
					m.err = err
					return nil
				}
			}
			m.form = nil
			m.state = usersStateList
			m.reloadList()
		case usersStateEditProfile:
			if m.editSave && m.selected != nil {
				if err := m.app.Users.UpdateProfile(m.selected.ID, m.editRealName, m.editLocation, m.editEmail); err != nil {
					m.err = err
					return nil
				}
			}
			m.refreshSelected()
			m.form = nil
			m.state = usersStateDetail
			m.list = newActionList(m.width, m.height)
		case usersStateResetPassword:
			if m.pwSave && m.selected != nil {
				if err := m.app.Users.UpdatePassword(m.selected.ID, m.newPassword); err != nil {
					m.err = err
					return nil
				}
			}
			m.form = nil
			m.state = usersStateDetail
			m.list = newActionList(m.width, m.height)
		case usersStateSetLevel:
			if m.levelSave && m.selected != nil {
				lvl, err := parseLevelChoice(m.levelChoice)
				if err != nil {
					m.err = err
					return nil
				}
				if err := m.app.Users.UpdateSecurityLevel(m.selected.ID, lvl); err != nil {
					m.err = err
					return nil
				}
			}
			m.refreshSelected()
			m.form = nil
			m.state = usersStateDetail
			m.list = newActionList(m.width, m.height)
		case usersStateSetANSI:
			if m.ansiSave && m.selected != nil {
				if err := m.app.Users.UpdateANSI(m.selected.ID, m.ansiEnabled); err != nil {
					m.err = err
					return nil
				}
			}
			m.refreshSelected()
			m.form = nil
			m.state = usersStateDetail
			m.list = newActionList(m.width, m.height)
		}
		return nil
	}
	return cmd
}

func (m *usersModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Users error: %v\n\nPress Enter/Esc to go back.", m.err)
	}

	switch m.state {
	case usersStateList:
		m.list.Title = "Users"
		return m.list.View() + "\n(q to quit, enter to select)"
	case usersStateDetail:
		if m.selected == nil {
			return "No user selected\n\n(esc to go back)"
		}
		header := fmt.Sprintf("User: %s (level %d)\n", m.selected.Username, m.selected.SecurityLevel)
		meta := fmt.Sprintf("Real name: %s\nLocation: %s\nEmail: %s\nANSI: %v\nTotal calls: %d\n\n",
			m.selected.RealName, m.selected.Location, m.selected.Email, m.selected.ANSIEnabled, m.selected.TotalCalls,
		)
		m.list.Title = "Actions"
		return header + meta + m.list.View() + "\n(esc to go back)"
	default:
		return m.form.View() + "\n\n(esc to go back)"
	}
}

func (m *usersModel) reloadList() {
	users, err := m.app.Users.List()
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0, len(users)+1)
	items = append(items, userItem{title: "+ Create new user", desc: "Add a new account", kind: "create"})
	for _, u := range users {
		desc := fmt.Sprintf("level %d • calls %d", u.SecurityLevel, u.TotalCalls)
		items = append(items, userItem{id: u.ID, title: u.Username, desc: desc, kind: "user"})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.SetShowHelp(true)
	m.list.Title = "Users"
}

func newActionList(w, h int) list.Model {
	items := []list.Item{
		userItem{title: "Edit profile", desc: "Real name, location, email", kind: "edit_profile"},
		userItem{title: "Set security level", desc: "New/Validated/Regular/Trusted/CoSysop/Sysop", kind: "set_level"},
		userItem{title: "Toggle ANSI", desc: "Enable/disable ANSI for user", kind: "set_ansi"},
		userItem{title: "Reset password", desc: "Set a new password", kind: "reset_password"},
		userItem{title: "Back", desc: "Return to users list", kind: "back"},
	}
	l := list.New(items, list.NewDefaultDelegate(), w, h-8)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	return l
}

func (m *usersModel) startCreate() {
	m.state = usersStateCreate
	m.createUsername = ""
	m.createPassword = ""
	m.createRealName = ""
	m.createLocation = ""
	m.createEmail = ""
	m.createSave = true
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Username").Value(&m.createUsername).Validate(nonEmpty("username")),
			huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&m.createPassword).Validate(nonEmpty("password")),
			huh.NewInput().Title("Real name").Value(&m.createRealName),
			huh.NewInput().Title("Location").Value(&m.createLocation),
			huh.NewInput().Title("Email").Value(&m.createEmail),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Create user?").Value(&m.createSave),
		),
	)
}

func (m *usersModel) startEditProfile() {
	m.state = usersStateEditProfile
	m.editRealName = m.selected.RealName
	m.editLocation = m.selected.Location
	m.editEmail = m.selected.Email
	m.editSave = true
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Real name").Value(&m.editRealName),
			huh.NewInput().Title("Location").Value(&m.editLocation),
			huh.NewInput().Title("Email").Value(&m.editEmail),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Save changes?").Value(&m.editSave),
		),
	)
}

func (m *usersModel) startResetPassword() {
	m.state = usersStateResetPassword
	m.newPassword = ""
	m.pwConfirm = ""
	m.pwSave = true
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("New password").EchoMode(huh.EchoModePassword).Value(&m.newPassword).Validate(nonEmpty("password")),
			huh.NewInput().Title("Confirm password").EchoMode(huh.EchoModePassword).Value(&m.pwConfirm).Validate(func(s string) error {
				if s != m.newPassword {
					return fmt.Errorf("passwords do not match")
				}
				return nil
			}),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Reset password?").Value(&m.pwSave),
		),
	)
}

func (m *usersModel) startSetLevel() {
	m.state = usersStateSetLevel
	m.levelChoice = fmt.Sprintf("%d", m.selected.SecurityLevel)
	m.levelSave = true
	options := []huh.Option[string]{
		huh.NewOption("New (10)", "new"),
		huh.NewOption("Validated (20)", "validated"),
		huh.NewOption("Regular (30)", "regular"),
		huh.NewOption("Trusted (50)", "trusted"),
		huh.NewOption("CoSysop (90)", "cosysop"),
		huh.NewOption("Sysop (100)", "sysop"),
		huh.NewOption("Custom (type number)", "custom"),
	}

	custom := ""
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Security level").Options(options...).Value(&m.levelChoice),
			huh.NewInput().Title("Custom level (only if selected)").Value(&custom).Validate(func(s string) error {
				if m.levelChoice != "custom" {
					return nil
				}
				_, err := strconv.Atoi(strings.TrimSpace(s))
				if err != nil {
					return fmt.Errorf("must be a number")
				}
				m.levelChoice = strings.TrimSpace(s)
				return nil
			}),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Save level?").Value(&m.levelSave),
		),
	)
}

func (m *usersModel) startSetANSI() {
	m.state = usersStateSetANSI
	m.ansiEnabled = m.selected.ANSIEnabled
	m.ansiSave = true
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("ANSI enabled").Value(&m.ansiEnabled),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Save ANSI preference?").Value(&m.ansiSave),
		),
	)
}

func (m *usersModel) back() {
	switch m.state {
	case usersStateList:
		m.Done = true
	case usersStateDetail:
		m.state = usersStateList
		m.selected = nil
		m.form = nil
		m.reloadList()
	default:
		m.state = usersStateDetail
		m.form = nil
		m.list = newActionList(m.width, m.height)
	}
}

func (m *usersModel) refreshSelected() {
	if m.selected == nil {
		return
	}
	u, err := m.app.Users.GetByID(m.selected.ID)
	if err == nil {
		m.selected = u
	}
}

func parseLevelChoice(choice string) (int, error) {
	s := strings.ToLower(strings.TrimSpace(choice))
	switch s {
	case "new":
		return user.LevelNew, nil
	case "validated":
		return user.LevelValidated, nil
	case "regular":
		return user.LevelRegular, nil
	case "trusted":
		return user.LevelTrusted, nil
	case "cosysop":
		return user.LevelCoSysop, nil
	case "sysop":
		return user.LevelSysop, nil
	default:
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("invalid level")
		}
		return v, nil
	}
}
