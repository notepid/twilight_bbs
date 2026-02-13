package node

import (
	"log"
	"time"

	"database/sql"

	"github.com/mikael/twilight_bbs/internal/ansi"
	"github.com/mikael/twilight_bbs/internal/chat"
	"github.com/mikael/twilight_bbs/internal/door"
	"github.com/mikael/twilight_bbs/internal/filearea"
	"github.com/mikael/twilight_bbs/internal/menu"
	"github.com/mikael/twilight_bbs/internal/message"
	"github.com/mikael/twilight_bbs/internal/terminal"
	"github.com/mikael/twilight_bbs/internal/user"
)

// Node represents a single BBS connection (one user session).
type Node struct {
	ID        int
	Term      *terminal.Terminal
	ConnectAt time.Time
	Remote    string

	// Menu state
	CurrentMenu string
	MenuStack   []string

	// Will be set after login
	UserID   int
	UserName string

	// Dependencies injected from main
	MenuRegistry *menu.Registry
	ANSILoader   *ansi.Loader
	UserRepo     *user.Repo
	MessageRepo  *message.Repo
	FileRepo     *filearea.Repo
	ChatBroker   *chat.Broker
	DoorLauncher *door.Launcher
	DB           *sql.DB

	// Shutdown signal
	done chan struct{}
}

// NewNode creates a new node for the given terminal.
func NewNode(id int, term *terminal.Terminal, remoteAddr string) *Node {
	return &Node{
		ID:        id,
		Term:      term,
		ConnectAt: time.Now(),
		Remote:    remoteAddr,
		done:      make(chan struct{}),
	}
}

// Run executes the main loop for this node.
func (n *Node) Run(mgr *Manager) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Node %d panic: %v", n.ID, r)
		}
		n.Term.Close()
		mgr.Remove(n.ID)
		log.Printf("Node %d disconnected (%s)", n.ID, n.Remote)
	}()

	log.Printf("Node %d connected from %s", n.ID, n.Remote)

	if n.MenuRegistry != nil && n.ANSILoader != nil {
		svc := &menu.Services{
			UserRepo:     n.UserRepo,
			MessageRepo:  n.MessageRepo,
			FileRepo:     n.FileRepo,
			ChatBroker:   n.ChatBroker,
			DoorLauncher: n.DoorLauncher,
			DB:           n.DB,
			NodeID:       n.ID,
		}

		engine := menu.NewEngine(n.MenuRegistry, n.ANSILoader, n.Term, svc)
		defer engine.Close()

		startMenu := "welcome"
		if m := n.MenuRegistry.Get(startMenu); m == nil {
			startMenu = "main_menu"
		}

		if err := engine.Run(startMenu); err != nil {
			log.Printf("Node %d menu engine error: %v", n.ID, err)
		}

		// Update node info from engine
		if u := engine.CurrentUser(); u != nil {
			n.UserID = u.ID
			n.UserName = u.Username
		}
	} else {
		n.Term.SendLn("Welcome to Twilight BBS!")
		n.Term.SendLn("Menu system not configured. Type Q to quit.")
		n.simpleLoop()
	}
}

func (n *Node) simpleLoop() error {
	for {
		key, err := n.Term.GetKey()
		if err != nil {
			return err
		}
		if key == 'Q' || key == 'q' {
			n.Term.SendLn("\r\nGoodbye!")
			return nil
		}
		n.Term.Send(string(key))
	}
}

// Disconnect closes the node connection.
func (n *Node) Disconnect() {
	select {
	case <-n.done:
	default:
		close(n.done)
	}
	n.Term.Close()
}
