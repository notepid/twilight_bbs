package menu

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"database/sql"

	"github.com/notepid/twilight_bbs/internal/ansi"
	"github.com/notepid/twilight_bbs/internal/chat"
	"github.com/notepid/twilight_bbs/internal/door"
	"github.com/notepid/twilight_bbs/internal/filearea"
	"github.com/notepid/twilight_bbs/internal/message"
	"github.com/notepid/twilight_bbs/internal/scripting"
	"github.com/notepid/twilight_bbs/internal/terminal"
	"github.com/notepid/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// ErrDisconnect is returned when the user disconnects.
var ErrDisconnect = errors.New("user disconnected")

// ErrMenuNotFound is returned when a menu is not found.
var ErrMenuNotFound = errors.New("menu not found")

// Services holds shared services available to the menu engine.
type Services struct {
	UserRepo     *user.Repo
	MessageRepo  *message.Repo
	FileRepo     *filearea.Repo
	ChatBroker   *chat.Broker
	DoorLauncher *door.Launcher
	DB           *sql.DB
	NodeID       int
}

// Engine manages the menu system for a single node/session.
type Engine struct {
	registry *Registry
	loader   *ansi.Loader
	term     *terminal.Terminal
	services *Services
	vm       *scripting.VM
	nodeAPI  *scripting.NodeAPI
	userAPI  *scripting.UserAPI
	msgAPI   *scripting.MessageAPI
	fileAPI  *scripting.FileAPI
	chatAPI  *scripting.ChatAPI
	doorAPI  *scripting.DoorAPI
	nodeUD   *lua.LUserData

	// Current user
	currentUser *user.User

	// Current state
	currentMenu string
	menuStack   []string
	running     bool

	// Navigation signals
	nextMenu   string
	gosubMenu  string
	returnMenu bool
	disconnect bool

	// Persistent menu state
	menuState map[string]map[string]interface{}
}

// NewEngine creates a new menu engine for a session.
func NewEngine(registry *Registry, loader *ansi.Loader, term *terminal.Terminal, svc *Services) *Engine {
	vm := scripting.NewVM()
	nodeAPI := scripting.NewNodeAPI(term)

	e := &Engine{
		registry:  registry,
		loader:    loader,
		term:      term,
		services:  svc,
		vm:        vm,
		nodeAPI:   nodeAPI,
		running:   true,
		menuState: make(map[string]map[string]interface{}),
	}

	// Wire navigation callbacks
	nodeAPI.OnGotoMenu = e.handleGotoMenu
	nodeAPI.OnGosubMenu = e.handleGosubMenu
	nodeAPI.OnReturnMenu = e.handleReturnMenu
	nodeAPI.OnDisconnect = e.handleDisconnect
	nodeAPI.OnDisplay = e.handleDisplay

	// Wire state callbacks
	nodeAPI.OnSetMenuState = e.SetMenuState
	nodeAPI.OnGetMenuState = e.GetMenuState

	// Register the node API in the Lua VM
	e.nodeUD = nodeAPI.Register(vm.L)

	// Register user API if repo is available
	if svc != nil && svc.UserRepo != nil {
		e.userAPI = scripting.NewUserAPI(svc.UserRepo)
		e.userAPI.OnLogin = e.handleUserLogin
		e.userAPI.Register(vm.L)
	}

	// Register message API if repo is available
	if svc != nil && svc.MessageRepo != nil {
		e.msgAPI = scripting.NewMessageAPI(svc.MessageRepo, func() *user.User {
			return e.currentUser
		})
		e.msgAPI.Register(vm.L)
	}

	// Register file API if repo is available
	if svc != nil && svc.FileRepo != nil {
		e.fileAPI = scripting.NewFileAPI(svc.FileRepo, func() *user.User {
			return e.currentUser
		})
		e.fileAPI.Register(vm.L)
	}

	// Register chat API if broker is available
	if svc != nil && svc.ChatBroker != nil {
		e.chatAPI = scripting.NewChatAPI(svc.ChatBroker, term, svc.NodeID, func() string {
			if e.currentUser != nil {
				return e.currentUser.Username
			}
			return fmt.Sprintf("Node %d", svc.NodeID)
		})
		e.chatAPI.Register(vm.L)

		// Wire inter-node callbacks
		nodeAPI.OnShowOnline = e.handleShowOnline
		nodeAPI.OnEnterChat = e.handleEnterChat
	}

	// Register door API if launcher is available
	if svc != nil && svc.DoorLauncher != nil && svc.DB != nil {
		e.doorAPI = scripting.NewDoorAPI(svc.DB, svc.DoorLauncher, func() *user.User {
			return e.currentUser
		}, svc.NodeID, term, term)
		e.doorAPI.Register(vm.L)

		nodeAPI.OnLaunchDoor = func(name string) error {
			e.term.SendLn(fmt.Sprintf("\r\n  Launching door: %s...", name))
			// The door API handles the actual launch through Lua
			return nil
		}
	}

	return e
}

// Close shuts down the menu engine.
func (e *Engine) Close() {
	e.vm.Close()
}

// CurrentUser returns the currently logged-in user, or nil.
func (e *Engine) CurrentUser() *user.User {
	return e.currentUser
}

// Run starts the menu engine at the given initial menu.
func (e *Engine) Run(startMenu string) error {
	e.currentMenu = startMenu

	for e.running {
		if err := e.runMenu(e.currentMenu); err != nil {
			if errors.Is(err, ErrDisconnect) {
				return nil
			}
			if errors.Is(err, ErrMenuNotFound) {
				// Try to recover by going to main menu
				if e.currentMenu != "main_menu" {
					e.currentMenu = "main_menu"
					continue
				}
				return err
			}
			return err
		}

		// Process navigation signals
		if e.disconnect {
			return nil
		}
		if e.nextMenu != "" {
			e.currentMenu = e.nextMenu
			e.nextMenu = ""
			continue
		}
		if e.gosubMenu != "" {
			e.menuStack = append(e.menuStack, e.currentMenu)
			e.currentMenu = e.gosubMenu
			e.gosubMenu = ""
			continue
		}
		if e.returnMenu {
			e.returnMenu = false
			if len(e.menuStack) > 0 {
				e.currentMenu = e.menuStack[len(e.menuStack)-1]
				e.menuStack = e.menuStack[:len(e.menuStack)-1]
				continue
			}
			return nil
		}
	}

	return nil
}

// CurrentMenuName returns the name of the current menu.
func (e *Engine) CurrentMenuName() string {
	return e.currentMenu
}

// runMenu loads and runs a single menu.
func (e *Engine) runMenu(name string) error {
	m := e.registry.Get(name)
	if m == nil {
		log.Printf("Menu not found: %s", name)
		e.term.SendLn(fmt.Sprintf("\r\nMenu '%s' not found.", name))
		e.term.Pause()
		return ErrMenuNotFound
	}

	// Load and run the Lua script
	if m.HasScript() {
		// Create a fresh VM for each menu to avoid state leakage
		oldVM := e.vm
		e.vm = scripting.NewVM()
		e.nodeUD = e.nodeAPI.Register(e.vm.L)
		e.nodeAPI.CurrentMenuName = name

		// Re-register APIs in the new VM
		if e.userAPI != nil {
			e.userAPI.Register(e.vm.L)
		}
		if e.msgAPI != nil {
			e.msgAPI.Register(e.vm.L)
		}
		if e.fileAPI != nil {
			e.fileAPI.Register(e.vm.L)
		}
		if e.chatAPI != nil {
			e.chatAPI.Register(e.vm.L)
		}
		if e.doorAPI != nil {
			e.doorAPI.Register(e.vm.L)
		}

		oldVM.Close()

		if err := e.vm.LoadScript(m.ScriptPath); err != nil {
			log.Printf("Script error in %s: %v", m.ScriptPath, err)
			e.term.SendLn(fmt.Sprintf("\r\nScript error: %v", err))
			e.term.Pause()
			// Continue to show menu even if script fails?
			// Maybe return nil to abort this menu but not the session?
			// For now let's continue to display part
		}
	} else {
		// Even if no script, we might want to close old VM?
		// Actually the existing code didn't close old VM if !m.HasScript(), which might be a bug or intentional to keep previous state?
		// But runMenu creates a NEW VM every time. So if we don't have a script, we are just pausing.
		// Let's stick to the previous logic but reorganized.
	}

	// Call on_load if script exists
	if m.HasScript() {
		if err := e.vm.CallMenuHandler("on_load", e.nodeUD); err != nil {
			scripting.LogError(name+".on_load", err)
		}
	}

	// Display the menu file
	displayPath := m.DisplayPath(e.term.ANSIEnabled)
	if displayPath != "" {
		df, err := e.loader.Load(displayPath)
		if err != nil {
			log.Printf("Failed to load display file %s: %v", displayPath, err)
		} else {
			if err := ansi.Display(e.term, df); err != nil {
				return fmt.Errorf("display menu %s: %w", name, err)
			}
		}
	}

	if !m.HasScript() {
		e.term.Pause()
		return nil
	}

	// Call on_enter
	if err := e.vm.CallMenuHandler("on_enter", e.nodeUD); err != nil {
		scripting.LogError(name+".on_enter", err)
	}

	// Check if on_enter already triggered navigation
	if e.hasNavigationPending() {
		// Call on_exit before leaving
		e.vm.CallMenuHandler("on_exit", e.nodeUD)
		return nil
	}

	// Enter the input loop
	if err := e.inputLoop(name); err != nil {
		return err
	}

	// Call on_exit
	if err := e.vm.CallMenuHandler("on_exit", e.nodeUD); err != nil {
		scripting.LogError(name+".on_exit", err)
	}

	return nil
}

// inputLoop reads input and dispatches to Lua handlers.
func (e *Engine) inputLoop(menuName string) error {
	hasOnKey := e.vm.HasMenuHandler("on_key")
	hasOnInput := e.vm.HasMenuHandler("on_input")

	if !hasOnKey && !hasOnInput {
		return nil
	}

	for e.running && !e.hasNavigationPending() {
		if hasOnKey {
			key, err := e.term.GetKey()
			if err != nil {
				return ErrDisconnect
			}

			keyStr := string(key)
			if err := e.vm.CallMenuHandler("on_key", e.nodeUD, lua.LString(keyStr)); err != nil {
				scripting.LogError(menuName+".on_key", err)
			}
		} else if hasOnInput {
			e.term.Send("> ")
			line, err := e.term.GetLine(80)
			if err != nil {
				return ErrDisconnect
			}

			line = strings.TrimSpace(line)
			if line != "" {
				if err := e.vm.CallMenuHandler("on_input", e.nodeUD, lua.LString(line)); err != nil {
					scripting.LogError(menuName+".on_input", err)
				}
			}
		}
	}

	return nil
}

// hasNavigationPending checks if a navigation signal has been set.
func (e *Engine) hasNavigationPending() bool {
	return e.nextMenu != "" || e.gosubMenu != "" || e.returnMenu || e.disconnect
}

// SetMenuState stores a value in the persistent state for a menu.
func (e *Engine) SetMenuState(menuName, key string, value interface{}) {
	if e.menuState[menuName] == nil {
		e.menuState[menuName] = make(map[string]interface{})
	}
	e.menuState[menuName][key] = value
}

// GetMenuState retrieves a value from the persistent state for a menu.
func (e *Engine) GetMenuState(menuName, key string) (interface{}, bool) {
	if e.menuState[menuName] == nil {
		return nil, false
	}
	val, ok := e.menuState[menuName][key]
	return val, ok
}

// --- Callbacks ---

func (e *Engine) handleGotoMenu(name string) error {
	e.nextMenu = name
	return nil
}

func (e *Engine) handleGosubMenu(name string) error {
	e.gosubMenu = name
	return nil
}

func (e *Engine) handleReturnMenu() error {
	e.returnMenu = true
	return nil
}

func (e *Engine) handleDisconnect() {
	e.disconnect = true
	e.running = false
}

func (e *Engine) handleDisplay(name string) error {
	df, err := e.loader.Find(name, e.term.ANSIEnabled)
	if err != nil {
		return err
	}
	return ansi.Display(e.term, df)
}

func (e *Engine) handleUserLogin(u *user.User) {
	e.currentUser = u
	// Update terminal ANSI setting based on user preference
	e.term.ANSIEnabled = u.ANSIEnabled
	if e.services != nil && e.services.ChatBroker != nil {
		e.services.ChatBroker.UpdateOnlineName(e.services.NodeID, u.Username)
	}
}

func (e *Engine) handleShowOnline() error {
	if e.services == nil || e.services.ChatBroker == nil {
		e.term.SendLn("\r\n  Who's online not available.")
		return nil
	}

	users := e.services.ChatBroker.ListOnline()
	e.term.SendLn("")
	e.term.SendLn("  Node  User               Status")
	e.term.SendLn("  ----  -----------------  --------")
	for _, u := range users {
		status := "Online"
		if u.Room != "" {
			status = "Chat: " + u.Room
		}
		e.term.SendLn(fmt.Sprintf("  %-4d  %-17s  %s", u.NodeID, u.UserName, status))
	}
	if len(users) == 0 {
		e.term.SendLn("  No users online.")
	}
	e.term.SendLn("")
	e.term.Pause()
	return nil
}

func (e *Engine) handleEnterChat() error {
	if e.services == nil || e.services.ChatBroker == nil {
		e.term.SendLn("\r\n  Chat not available.")
		return nil
	}

	room := "main"
	broker := e.services.ChatBroker
	nodeID := e.services.NodeID
	userName := "Unknown"
	if e.currentUser != nil {
		userName = e.currentUser.Username
	}

	// Subscribe to chat
	sub := broker.Subscribe(nodeID, userName)

	// Join room
	broker.JoinRoom(nodeID, room)

	// Announce arrival
	broker.SendToRoom(nodeID, userName, room,
		fmt.Sprintf("*** %s has joined ***", userName))

	e.term.Cls()
	e.term.SendLn("  Chat Room: " + room)
	e.term.SendLn("  Type /quit to leave, /who to see users")
	e.term.SendLn("  ─────────────────────────────────────")
	e.term.SendLn("")

	// Simple chat loop - read messages and input
	// We use a goroutine to display incoming messages
	done := make(chan struct{})
	go func() {
		defer func() { recover() }()
		for {
			select {
			case msg, ok := <-sub.Ch:
				if !ok {
					return
				}
				e.term.SendLn(fmt.Sprintf("\r  <%s> %s", msg.FromUser, msg.Text))
			case <-done:
				return
			}
		}
	}()

	cleanup := func() {
		select {
		case <-done:
			// already closed
		default:
			close(done)
		}
		broker.LeaveRoom(nodeID)
		broker.Unsubscribe(nodeID)
	}

	for {
		line, err := e.term.GetLine(200)
		if err != nil {
			cleanup()
			break
		}

		if line == "/quit" || line == "/q" {
			broker.SendToRoom(nodeID, userName, room,
				fmt.Sprintf("*** %s has left ***", userName))
			cleanup()
			break
		}

		if line == "/who" {
			members := broker.RoomMembers(room)
			e.term.SendLn("  Users in room: " + fmt.Sprintf("%v", members))
			continue
		}

		if line != "" {
			// Send to room
			broker.SendToRoom(nodeID, userName, room, line)
			// Echo locally
			e.term.SendLn(fmt.Sprintf("  <%s> %s", userName, line))
		}
	}
	cleanup()

	e.term.SendLn("")
	e.term.SendLn("  Left chat room.")
	if !e.hasNavigationPending() {
		e.nextMenu = e.currentMenu
	}
	return nil
}
