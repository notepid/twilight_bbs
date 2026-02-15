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

type filesModel struct {
	app *app.App

	width  int
	height int

	Done bool

	state filesState
	list  list.Model
	err   error

	selectedAreaID int
	offset         int
	limit          int

	selectedFileID int
	fileDetail     string

	form *huh.Form

	searchQuery string

	addFilename    string
	addDescription string
	addSizeBytes   string
	addUploader    string
	addSave        bool
}

type filesState int

const (
	filesStateAreas filesState = iota
	filesStateList
	filesStateDetail
	filesStateSearch
	filesStateAddEntry
)

type fileItem struct {
	id    int
	title string
	desc  string
	kind  string
}

func (i fileItem) Title() string       { return i.title }
func (i fileItem) Description() string { return i.desc }
func (i fileItem) FilterValue() string { return i.title }

func newFilesModel(a *app.App) *filesModel {
	m := &filesModel{app: a, state: filesStateAreas, limit: 50}
	m.reloadAreas()
	return m
}

func (m *filesModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w, h-2)
}

func (m *filesModel) Update(msg tea.Msg) tea.Cmd {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.err = nil
				m.state = filesStateAreas
				m.reloadAreas()
			}
		}
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.state == filesStateAreas {
				m.Done = true
				return nil
			}
		case "esc":
			m.back()
			return nil
		case "n":
			if m.state == filesStateList {
				m.offset += m.limit
				m.reloadFiles()
				return nil
			}
		case "p":
			if m.state == filesStateList {
				m.offset -= m.limit
				if m.offset < 0 {
					m.offset = 0
				}
				m.reloadFiles()
				return nil
			}
		case "/":
			if m.state == filesStateList {
				m.startSearch()
				return nil
			}
		case "a":
			if m.state == filesStateList {
				m.startAddEntry()
				return nil
			}
		}
	}

	if m.state == filesStateSearch || m.state == filesStateAddEntry {
		return m.updateForm(msg)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			it, ok := m.list.SelectedItem().(fileItem)
			if !ok {
				return cmd
			}
			switch m.state {
			case filesStateAreas:
				m.selectedAreaID = it.id
				m.offset = 0
				m.state = filesStateList
				m.reloadFiles()
				return nil
			case filesStateList:
				m.selectedFileID = it.id
				m.state = filesStateDetail
				m.loadFileDetail()
				return nil
			}
		}
	}

	return cmd
}

func (m *filesModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Files error: %v\n\nPress Enter/Esc to go back.", m.err)
	}

	switch m.state {
	case filesStateAreas:
		m.list.Title = "File Areas"
		return m.list.View() + "\n(q to quit, enter to select)"
	case filesStateList:
		m.list.Title = fmt.Sprintf("Files (area %d)", m.selectedAreaID)
		return m.list.View() + "\n(n next page, p prev page, / search, a add entry, esc back)"
	case filesStateDetail:
		return m.fileDetail + "\n\n(esc back)"
	case filesStateSearch, filesStateAddEntry:
		return m.form.View() + "\n\n(esc back)"
	default:
		return "Files"
	}
}

func (m *filesModel) reloadAreas() {
	areas, err := m.app.Files.ListAreas(user.LevelSysop)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0, len(areas))
	for _, a := range areas {
		desc := fmt.Sprintf("%s • %d files", a.Description, a.FileCount)
		items = append(items, fileItem{id: a.ID, title: a.Name, desc: desc, kind: "area"})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.SetShowHelp(true)
}

func (m *filesModel) reloadFiles() {
	files, err := m.app.Files.ListFiles(m.selectedAreaID, m.offset, m.limit)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		desc := fmt.Sprintf("%d bytes • %d dl", f.SizeBytes, f.DownloadCount)
		items = append(items, fileItem{id: f.ID, title: f.Filename, desc: desc, kind: "file"})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(true)
	m.list.SetShowHelp(true)
}

func (m *filesModel) loadFileDetail() {
	f, err := m.app.Files.GetFile(m.selectedFileID)
	if err != nil {
		m.err = err
		return
	}

	m.fileDetail = fmt.Sprintf("File: %s\nDescription: %s\nSize: %d bytes\nUploader: %s\nDownloads: %d\nUploaded: %s",
		f.Filename, f.Description, f.SizeBytes, f.UploaderName, f.DownloadCount, f.UploadedAt.Format("2006-01-02 15:04"),
	)
}

func (m *filesModel) startSearch() {
	m.state = filesStateSearch
	m.searchQuery = ""
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Search files (name/description)").Value(&m.searchQuery).Validate(nonEmpty("query")),
		),
	)
}

func (m *filesModel) startAddEntry() {
	m.state = filesStateAddEntry
	m.addFilename = ""
	m.addDescription = ""
	m.addSizeBytes = "0"
	m.addUploader = ""
	m.addSave = true
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Filename").Value(&m.addFilename).Validate(nonEmpty("filename")),
			huh.NewInput().Title("Description").Value(&m.addDescription),
			huh.NewInput().Title("Size bytes").Value(&m.addSizeBytes).Validate(validIntGreaterThan("size bytes", -1)),
			huh.NewInput().Title("Uploader username").Value(&m.addUploader).Validate(nonEmpty("uploader")),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Add entry?").Value(&m.addSave),
		),
	)
}

func (m *filesModel) updateForm(msg tea.Msg) tea.Cmd {
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
		case filesStateSearch:
			results, err := m.app.Files.FindByName(m.searchQuery, user.LevelSysop)
			if err != nil {
				m.err = err
				return nil
			}
			items := make([]list.Item, 0, len(results))
			for _, f := range results {
				desc := fmt.Sprintf("area %d • %d bytes", f.AreaID, f.SizeBytes)
				items = append(items, fileItem{id: f.ID, title: f.Filename, desc: desc, kind: "file"})
			}
			m.state = filesStateList
			m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height-2)
			m.list.SetShowStatusBar(false)
			m.list.SetFilteringEnabled(true)
			m.list.SetShowHelp(true)
			m.form = nil
			return nil
		case filesStateAddEntry:
			if m.addSave {
				u, err := m.app.Users.GetByUsername(strings.TrimSpace(m.addUploader))
				if err != nil {
					m.err = err
					return nil
				}
				sz, _ := strconv.ParseInt(strings.TrimSpace(m.addSizeBytes), 10, 64)
				_, err = m.app.Files.AddEntry(m.selectedAreaID, strings.TrimSpace(m.addFilename), strings.TrimSpace(m.addDescription), sz, u.ID)
				if err != nil {
					m.err = err
					return nil
				}
			}
			m.form = nil
			m.state = filesStateList
			m.reloadFiles()
			return nil
		}
	}
	return cmd
}

func (m *filesModel) back() {
	switch m.state {
	case filesStateAreas:
		m.Done = true
	case filesStateList:
		m.state = filesStateAreas
		m.reloadAreas()
	case filesStateDetail:
		m.state = filesStateList
		m.reloadFiles()
	case filesStateSearch, filesStateAddEntry:
		m.form = nil
		m.state = filesStateList
		m.reloadFiles()
	}
}
