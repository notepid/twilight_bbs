package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"

	"github.com/notepid/twilight_bbs/internal/admin/app"
	"github.com/notepid/twilight_bbs/internal/user"
)

type messagesModel struct {
	app *app.App

	width  int
	height int

	Done bool

	state messagesState
	list  list.Model
	err   error

	selectedAreaID int
	offset         int
	limit          int

	selectedMsgID int
	msgBody       string
	msgHeader     string
}

type messagesState int

const (
	messagesStateAreas messagesState = iota
	messagesStateList
	messagesStateDetail
)

type msgItem struct {
	id    int
	title string
	desc  string
	kind  string
}

func (i msgItem) Title() string       { return i.title }
func (i msgItem) Description() string { return i.desc }
func (i msgItem) FilterValue() string { return i.title }

func newMessagesModel(a *app.App) *messagesModel {
	m := &messagesModel{app: a, state: messagesStateAreas, limit: 50}
	m.reloadAreas()
	return m
}

func (m *messagesModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w, h-2)
}

func (m *messagesModel) Update(msg tea.Msg) tea.Cmd {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.err = nil
				m.state = messagesStateAreas
				m.reloadAreas()
			}
		}
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.state == messagesStateAreas {
				m.Done = true
				return nil
			}
		case "esc":
			m.back()
			return nil
		case "n":
			if m.state == messagesStateList {
				m.offset += m.limit
				m.reloadMessages()
				return nil
			}
		case "p":
			if m.state == messagesStateList {
				m.offset -= m.limit
				if m.offset < 0 {
					m.offset = 0
				}
				m.reloadMessages()
				return nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			it, ok := m.list.SelectedItem().(msgItem)
			if !ok {
				return cmd
			}
			switch m.state {
			case messagesStateAreas:
				m.selectedAreaID = it.id
				m.offset = 0
				m.state = messagesStateList
				m.reloadMessages()
				return nil
			case messagesStateList:
				m.selectedMsgID = it.id
				m.state = messagesStateDetail
				m.loadMessageDetail()
				return nil
			}
		}
	}

	return cmd
}

func (m *messagesModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Messages error: %v\n\nPress Enter/Esc to go back.", m.err)
	}

	switch m.state {
	case messagesStateAreas:
		m.list.Title = "Message Areas"
		return m.list.View() + "\n(q to quit, enter to select)"
	case messagesStateList:
		m.list.Title = fmt.Sprintf("Messages (area %d)", m.selectedAreaID)
		return m.list.View() + "\n(n next page, p prev page, esc back)"
	case messagesStateDetail:
		return m.msgHeader + "\n\n" + m.msgBody + "\n\n(esc back)"
	default:
		return "Messages"
	}
}

func (m *messagesModel) reloadAreas() {
	areas, err := m.app.Messages.ListAreas(user.LevelSysop)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0, len(areas))
	for _, a := range areas {
		desc := fmt.Sprintf("%s • total %d", a.Description, a.TotalMsgs)
		items = append(items, msgItem{id: a.ID, title: a.Name, desc: desc, kind: "area"})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.SetShowHelp(true)
}

func (m *messagesModel) reloadMessages() {
	msgs, err := m.app.Messages.ListMessages(m.selectedAreaID, m.offset, m.limit)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0, len(msgs))
	for _, msg := range msgs {
		desc := fmt.Sprintf("from %s to %s", msg.FromName, msg.ToName)
		items = append(items, msgItem{id: msg.ID, title: msg.Subject, desc: desc, kind: "msg"})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.SetShowHelp(true)
}

func (m *messagesModel) loadMessageDetail() {
	msg, err := m.app.Messages.GetMessage(m.selectedMsgID)
	if err != nil {
		m.err = err
		return
	}

	m.msgHeader = fmt.Sprintf("Subject: %s\nFrom: %s\nTo: %s\nDate: %s",
		msg.Subject, msg.FromName, msg.ToName, msg.CreatedAt.Format("2006-01-02 15:04"),
	)
	m.msgBody = msg.Body
}

func (m *messagesModel) back() {
	switch m.state {
	case messagesStateAreas:
		m.Done = true
	case messagesStateList:
		m.state = messagesStateAreas
		m.reloadAreas()
	case messagesStateDetail:
		m.state = messagesStateList
		m.reloadMessages()
	}
}
