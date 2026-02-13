package chat

import (
	"fmt"
	"log"
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
	online      map[int]*OnlineUser
}

// NewBroker creates a new chat message broker.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[int]*Subscriber),
		online:      make(map[int]*OnlineUser),
	}
}

// OnlineUser represents a connected user (regardless of chat participation).
type OnlineUser struct {
	NodeID   int
	UserName string
	Room     string
}

// RegisterOnline marks a node as connected.
func (b *Broker) RegisterOnline(nodeID int, userName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.online[nodeID] = &OnlineUser{
		NodeID:   nodeID,
		UserName: userName,
		Room:     "",
	}
}

// UpdateOnlineName updates the displayed username for a connected node.
func (b *Broker) UpdateOnlineName(nodeID int, userName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	u, ok := b.online[nodeID]
	if !ok {
		b.online[nodeID] = &OnlineUser{NodeID: nodeID, UserName: userName}
		return
	}
	u.UserName = userName
}

// UnregisterOnline removes a node from the online list.
func (b *Broker) UnregisterOnline(nodeID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.online, nodeID)
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
	// Ensure the node is visible in the online list.
	if u, ok := b.online[nodeID]; ok {
		u.UserName = userName
	} else {
		b.online[nodeID] = &OnlineUser{NodeID: nodeID, UserName: userName}
	}
	return sub
}

// Unsubscribe removes a node from the chat system.
func (b *Broker) Unsubscribe(nodeID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Don't close the channel here: broadcasters may have already snapshotted
	// subscribers and will send concurrently.
	delete(b.subscribers, nodeID)
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
		return nil
	default:
		return fmt.Errorf("node %d message buffer full", toNodeID)
	}
}

// Broadcast sends a message to all subscribed nodes except the sender.
func (b *Broker) Broadcast(fromNodeID int, fromUser string, text string) {
	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subscribers))
	for id, sub := range b.subscribers {
		if id == fromNodeID {
			continue
		}
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	msg := Message{
		FromNodeID: fromNodeID,
		FromUser:   fromUser,
		Text:       text,
	}

	dropped := 0
	for _, sub := range subs {
		select {
		case sub.Ch <- msg:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("chat: dropped %d broadcast messages (slow subscribers)", dropped)
	}
}

// SendToRoom sends a message to all nodes in a specific chat room.
func (b *Broker) SendToRoom(fromNodeID int, fromUser string, room string, text string) {
	b.mu.RLock()
	subs := make([]*Subscriber, 0, len(b.subscribers))
	for id, sub := range b.subscribers {
		if id == fromNodeID {
			continue
		}
		if sub.Room == room {
			subs = append(subs, sub)
		}
	}
	b.mu.RUnlock()

	msg := Message{
		FromNodeID: fromNodeID,
		FromUser:   fromUser,
		ToNodeID:   -1,
		Room:       room,
		Text:       text,
	}

	dropped := 0
	for _, sub := range subs {
		select {
		case sub.Ch <- msg:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("chat: dropped %d room messages (room=%q)", dropped, room)
	}
}

// JoinRoom puts a subscriber in a chat room.
func (b *Broker) JoinRoom(nodeID int, room string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[nodeID]; ok {
		sub.Room = room
	}
	if u, ok := b.online[nodeID]; ok {
		u.Room = room
	}
}

// LeaveRoom removes a subscriber from their current room.
func (b *Broker) LeaveRoom(nodeID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[nodeID]; ok {
		sub.Room = ""
	}
	if u, ok := b.online[nodeID]; ok {
		u.Room = ""
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

// ListOnline returns all currently connected users.
func (b *Broker) ListOnline() []OnlineUser {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var users []OnlineUser
	for _, u := range b.online {
		users = append(users, OnlineUser{
			NodeID:   u.NodeID,
			UserName: u.UserName,
			Room:     u.Room,
		})
	}
	return users
}
