package chat

import (
	"fmt"
	"strings"
	"sync"

	"github.com/notepid/twilight_bbs/internal/ansi"
	"github.com/notepid/twilight_bbs/internal/terminal"
)

// RoomSessionConfig configures a simple interactive chat session in a room.
type RoomSessionConfig struct {
	Term     *terminal.Terminal
	Broker   *Broker
	NodeID   int
	UserName string
	Room     string

	// Template is an optional display file (e.g. assets/menus/chat_room.asc)
	// that defines placeholder regions for the chat UI.
	//
	// If nil (or if ANSI is disabled), the session falls back to the classic
	// sequential chat output.
	Template *ansi.DisplayFile
}

// RunRoomSession runs a simple interactive chat session against the Broker.
// It handles subscribing, joining/leaving the room, and displaying incoming/outgoing messages.
func RunRoomSession(cfg RoomSessionConfig) error {
	if cfg.Term == nil || cfg.Broker == nil {
		return nil
	}
	if cfg.Room == "" {
		cfg.Room = "main"
	}
	if cfg.UserName == "" {
		cfg.UserName = "Unknown"
	}

	// If ANSI is disabled or we don't have a template, use the classic chat flow.
	if cfg.Template == nil || !cfg.Term.ANSIEnabled {
		return runSimpleRoomSession(cfg)
	}

	broker := cfg.Broker
	nodeID := cfg.NodeID
	userName := cfg.UserName
	room := cfg.Room

	// Subscribe to chat.
	sub := broker.Subscribe(nodeID, userName)

	// Join room.
	broker.JoinRoom(nodeID, room)

	// Announce arrival.
	broker.SendToRoom(nodeID, userName, room,
		fmt.Sprintf("*** %s has joined ***", userName))

	ui, ok := newTemplatedRoomUI(cfg.Term, cfg.Template)
	if !ok {
		// Template was missing required placeholders; fall back gracefully.
		return runSimpleRoomSession(cfg)
	}

	_ = cfg.Term.Cls()
	_ = ansi.Display(cfg.Term, cfg.Template)
	ui.outputField("ROOM", room)
	ui.outputField("STATUS", "Type /quit to leave, /who to list users")
	ui.appendSystem(fmt.Sprintf("*** Joined room: %s ***", room))

	done := make(chan struct{})
	go func() {
		defer func() { recover() }()
		for {
			select {
			case msg, ok := <-sub.Ch:
				if !ok {
					return
				}
				ui.appendMessage(msg.FromUser, msg.Text)
			case <-done:
				return
			}
		}
	}()

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(done)
			broker.LeaveRoom(nodeID)
			broker.Unsubscribe(nodeID)
		})
	}
	defer cleanup()

	for {
		line, err := ui.readInputLine()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "/quit" || line == "/q" {
			broker.SendToRoom(nodeID, userName, room,
				fmt.Sprintf("*** %s has left ***", userName))
			break
		}

		if line == "/who" {
			members := broker.RoomMembers(room)
			ui.appendSystem(fmt.Sprintf("*** Users in room: %s ***", strings.Join(members, ", ")))
			continue
		}

		if line != "" {
			// Send to room.
			broker.SendToRoom(nodeID, userName, room, line)
			// Echo locally.
			ui.appendMessage(userName, line)
		}
	}

	ui.outputField("STATUS", "Left chat room.")
	return nil
}

func runSimpleRoomSession(cfg RoomSessionConfig) error {
	broker := cfg.Broker
	nodeID := cfg.NodeID
	userName := cfg.UserName
	room := cfg.Room

	// Subscribe to chat.
	sub := broker.Subscribe(nodeID, userName)
	// Join room.
	broker.JoinRoom(nodeID, room)
	// Announce arrival.
	broker.SendToRoom(nodeID, userName, room,
		fmt.Sprintf("*** %s has joined ***", userName))

	_ = cfg.Term.Cls()
	_ = cfg.Term.SendLn("  Chat Room: " + room)
	_ = cfg.Term.SendLn("  Type /quit to leave, /who to see users")
	_ = cfg.Term.SendLn("  ---------------------------------------------")
	_ = cfg.Term.SendLn("")

	done := make(chan struct{})
	go func() {
		defer func() { recover() }()
		for {
			select {
			case msg, ok := <-sub.Ch:
				if !ok {
					return
				}
				_ = cfg.Term.SendLn(fmt.Sprintf("\r<%s> %s", msg.FromUser, msg.Text))
			case <-done:
				return
			}
		}
	}()

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(done)
			broker.LeaveRoom(nodeID)
			broker.Unsubscribe(nodeID)
		})
	}
	defer cleanup()

	for {
		line, err := cfg.Term.GetLine(200)
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		if line == "/quit" || line == "/q" {
			broker.SendToRoom(nodeID, userName, room,
				fmt.Sprintf("*** %s has left ***", userName))
			break
		}

		if line == "/who" {
			members := broker.RoomMembers(room)
			_ = cfg.Term.SendLn("  Users in room: " + fmt.Sprintf("%v", members))
			continue
		}

		if line != "" {
			broker.SendToRoom(nodeID, userName, room, line)
			_ = cfg.Term.SendLn(fmt.Sprintf("<%s> %s", userName, line))
		}
	}

	_ = cfg.Term.SendLn("")
	_ = cfg.Term.SendLn("  Left chat room.")
	return nil
}

type templatedRoomUI struct {
	term   *terminal.Terminal
	fields map[string]ansi.Field

	logWidth  int
	logHeight int

	mu   sync.Mutex
	logs []string

	// input holds the current user-typed buffer (for redraw during async output).
	input []byte
}

func newTemplatedRoomUI(term *terminal.Terminal, df *ansi.DisplayFile) (*templatedRoomUI, bool) {
	if term == nil || df == nil {
		return nil, false
	}

	width := term.Width
	if df.Sauce != nil && df.Sauce.TInfo1 > 0 {
		width = int(df.Sauce.TInfo1)
	}
	if width <= 0 {
		width = 80
	}

	fields := ansi.IndexFields(df, width)

	logF, ok := fields["CHAT_LOG"]
	if !ok || logF.MaxLen <= 0 || logF.Height <= 0 {
		return nil, false
	}
	inputF, ok := fields["INPUT"]
	if !ok || inputF.MaxLen <= 0 {
		return nil, false
	}

	return &templatedRoomUI{
		term:      term,
		fields:    fields,
		logWidth:  logF.MaxLen,
		logHeight: logF.Height,
		logs:      make([]string, 0, logF.Height),
	}, true
}

func (ui *templatedRoomUI) outputField(id, text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	f, ok := ui.fields[id]
	if !ok || f.Row <= 0 || f.Col <= 0 {
		return
	}

	width := f.MaxLen
	if width <= 0 {
		width = 80
	}
	height := f.Height
	if height <= 0 {
		height = 1
	}

	// Clear rectangle first.
	for row := 0; row < height; row++ {
		_ = ui.term.GotoXY(f.Row+row, f.Col)
		_ = ui.term.Send(strings.Repeat(" ", width))
	}

	// Print text into the rectangle (no wrapping, clip by width/height).
	lines := strings.Split(text, "\n")
	for i := 0; i < height && i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		r := []rune(line)
		if len(r) > width {
			line = string(r[:width])
		}
		_ = ui.term.GotoXY(f.Row+i, f.Col)
		_ = ui.term.Send(line)
	}
}

func (ui *templatedRoomUI) appendSystem(text string) {
	ui.appendLogLines(strings.Split(text, "\n"))
}

func (ui *templatedRoomUI) appendMessage(fromUser, text string) {
	ui.appendLogLines([]string{fmt.Sprintf("<%s> %s", fromUser, text)})
}

func (ui *templatedRoomUI) appendLogLines(lines []string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		ui.logs = append(ui.logs, line)
		if len(ui.logs) > ui.logHeight {
			ui.logs = ui.logs[len(ui.logs)-ui.logHeight:]
		}
	}

	ui.redrawLogLocked()
	ui.redrawInputLocked()
}

func (ui *templatedRoomUI) redrawLogLocked() {
	logF, ok := ui.fields["CHAT_LOG"]
	if !ok {
		return
	}

	// Clear log rectangle.
	for row := 0; row < ui.logHeight; row++ {
		_ = ui.term.GotoXY(logF.Row+row, logF.Col)
		_ = ui.term.Send(strings.Repeat(" ", ui.logWidth))
	}

	// Print buffered lines.
	for i := 0; i < ui.logHeight && i < len(ui.logs); i++ {
		line := ui.logs[i]
		r := []rune(line)
		if len(r) > ui.logWidth {
			line = string(r[:ui.logWidth])
		}
		_ = ui.term.GotoXY(logF.Row+i, logF.Col)
		_ = ui.term.Send(line)
	}
}

func (ui *templatedRoomUI) redrawInputLocked() {
	inputF, ok := ui.fields["INPUT"]
	if !ok || inputF.MaxLen <= 0 {
		return
	}

	_ = ui.term.GotoXY(inputF.Row, inputF.Col)
	_ = ui.term.Send(strings.Repeat(" ", inputF.MaxLen))
	_ = ui.term.GotoXY(inputF.Row, inputF.Col)
	if len(ui.input) > 0 {
		// Clip to field width.
		buf := ui.input
		if len(buf) > inputF.MaxLen {
			buf = buf[len(buf)-inputF.MaxLen:]
		}
		_ = ui.term.Send(string(buf))
	}
}

func (ui *templatedRoomUI) readInputLine() (string, error) {
	inputF, ok := ui.fields["INPUT"]
	if !ok {
		return "", nil
	}

	ui.mu.Lock()
	ui.input = ui.input[:0]
	_ = ui.term.GotoXY(inputF.Row, inputF.Col)
	_ = ui.term.Send(strings.Repeat(" ", inputF.MaxLen))
	_ = ui.term.GotoXY(inputF.Row, inputF.Col)
	ui.mu.Unlock()

	// Read input in-place (no CRLF emission), while allowing async log redraws.
	// All terminal writes are synchronized on ui.mu to avoid interleaved output.
	var buf []byte
	maxLen := inputF.MaxLen
	if maxLen <= 0 {
		maxLen = 200
	}
	for {
		b, err := ui.term.ReadByte()
		if err != nil {
			return string(buf), err
		}

		switch b {
		case '\r', '\n':
			// Submit without moving the cursor (the template owns layout).
			ui.mu.Lock()
			ui.input = ui.input[:0]
			ui.redrawInputLocked()
			ui.mu.Unlock()
			return string(buf), nil
		case 8, 127: // backspace or delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				ui.mu.Lock()
				if len(ui.input) > 0 {
					ui.input = ui.input[:len(ui.input)-1]
				}
				// Erase the last character visually.
				_ = ui.term.Send("\b \b")
				ui.mu.Unlock()
			}
		default:
			if b >= 32 && b < 127 && len(buf) < maxLen {
				buf = append(buf, b)
				ui.mu.Lock()
				ui.input = append(ui.input, b)
				_ = ui.term.Send(string(b))
				ui.mu.Unlock()
			}
		}
	}
}

// (No separate getLineInPlace helper; the UI synchronizes terminal output to
// avoid interleaving async chat updates with user-typed echo.)
