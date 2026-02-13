package chat

import (
	"fmt"
	"sync"
)

// Terminal is the minimal terminal interface needed to run a chat session.
// *terminal.Terminal satisfies this interface.
type Terminal interface {
	Cls() error
	SendLn(s string) error
	GetLine(maxLen int) (string, error)
}

// RoomSessionConfig configures a simple interactive chat session in a room.
type RoomSessionConfig struct {
	Term     Terminal
	Broker   *Broker
	NodeID   int
	UserName string
	Room     string
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
				_ = cfg.Term.SendLn(fmt.Sprintf("\r  <%s> %s", msg.FromUser, msg.Text))
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
			// Send to room.
			broker.SendToRoom(nodeID, userName, room, line)
			// Echo locally.
			_ = cfg.Term.SendLn(fmt.Sprintf("  <%s> %s", userName, line))
		}
	}

	_ = cfg.Term.SendLn("")
	_ = cfg.Term.SendLn("  Left chat room.")
	return nil
}
