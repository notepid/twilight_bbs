package chat

import (
	"fmt"
	"sync"
)

// Message represents a chat message.
type Message struct {
	FromNodeID int
	FromUser   string
	ToNodeID   int    // 0 = broadcast, -1 = room
	Room       string // room name if ToNodeID == -1
	Text       string
}

// Subscriber receives chat messages.
type Subscriber struct {
	NodeID   int
	UserName string
	Ch       chan Message
	Room     string // current chat room ("" = not in a room)
}

// Broker routes messages between nodes.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[int]*Subscriber
}

// NewBroker creates a new chat message broker.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[int]*Subscriber),
	}
}

// Subscribe registers a node to receive chat messages.
func (b *Broker) Subscribe(nodeID int, userName string) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &Subscriber{
		NodeID:   nodeID,
		UserName: userName,
		Ch:       make(chan Message, 32),
	}
	b.subscribers[nodeID] = sub
	return sub
}

// Unsubscribe removes a node from the chat system.
func (b *Broker) Unsubscribe(nodeID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[nodeID]; ok {
		close(sub.Ch)
		delete(b.subscribers, nodeID)
	}
}

// SendTo sends a message to a specific node.
func (b *Broker) SendTo(fromNodeID int, fromUser string, toNodeID int, text string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sub, ok := b.subscribers[toNodeID]
	if !ok {
		return fmt.Errorf("node %d not found", toNodeID)
	}

	msg := Message{
		FromNodeID: fromNodeID,
		FromUser:   fromUser,
		ToNodeID:   toNodeID,
		Text:       text,
	}

	select {
	case sub.Ch <- msg:
	default:
		// Buffer full, drop message
	}

	return nil
}

// Broadcast sends a message to all subscribed nodes except the sender.
func (b *Broker) Broadcast(fromNodeID int, fromUser string, text string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	msg := Message{
		FromNodeID: fromNodeID,
		FromUser:   fromUser,
		Text:       text,
	}

	for id, sub := range b.subscribers {
		if id == fromNodeID {
			continue
		}
		select {
		case sub.Ch <- msg:
		default:
		}
	}
}

// SendToRoom sends a message to all nodes in a specific chat room.
func (b *Broker) SendToRoom(fromNodeID int, fromUser string, room string, text string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	msg := Message{
		FromNodeID: fromNodeID,
		FromUser:   fromUser,
		ToNodeID:   -1,
		Room:       room,
		Text:       text,
	}

	for id, sub := range b.subscribers {
		if id == fromNodeID {
			continue
		}
		if sub.Room == room {
			select {
			case sub.Ch <- msg:
			default:
			}
		}
	}
}

// JoinRoom puts a subscriber in a chat room.
func (b *Broker) JoinRoom(nodeID int, room string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[nodeID]; ok {
		sub.Room = room
	}
}

// LeaveRoom removes a subscriber from their current room.
func (b *Broker) LeaveRoom(nodeID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[nodeID]; ok {
		sub.Room = ""
	}
}

// RoomMembers returns the usernames of all nodes in a room.
func (b *Broker) RoomMembers(room string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var members []string
	for _, sub := range b.subscribers {
		if sub.Room == room {
			members = append(members, sub.UserName)
		}
	}
	return members
}

// OnlineUsers returns a list of all subscribed users.
type OnlineUser struct {
	NodeID   int
	UserName string
	Room     string
}

// ListOnline returns all currently subscribed users.
func (b *Broker) ListOnline() []OnlineUser {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var users []OnlineUser
	for _, sub := range b.subscribers {
		users = append(users, OnlineUser{
			NodeID:   sub.NodeID,
			UserName: sub.UserName,
			Room:     sub.Room,
		})
	}
	return users
}
