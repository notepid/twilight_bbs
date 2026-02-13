package node

import (
	"fmt"
	"sync"
)

// Manager tracks all active nodes and enforces the max-nodes limit.
type Manager struct {
	mu       sync.RWMutex
	nodes    map[int]*Node
	maxNodes int
	nextID   int
	BBSName  string
	SysopName string
}

// NewManager creates a new node manager.
func NewManager(maxNodes int, bbsName, sysopName string) *Manager {
	return &Manager{
		nodes:     make(map[int]*Node),
		maxNodes:  maxNodes,
		nextID:    1,
		BBSName:   bbsName,
		SysopName: sysopName,
	}
}

// Acquire allocates a node ID if capacity allows.
// Returns the node ID and true, or 0 and false if full.
func (m *Manager) Acquire() (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.nodes) >= m.maxNodes {
		return 0, false
	}

	id := m.nextID
	m.nextID++
	return id, true
}

// Add registers a node with the manager.
func (m *Manager) Add(n *Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[n.ID] = n
}

// Remove removes a node from the manager.
func (m *Manager) Remove(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodes, id)
}

// Get returns a node by ID, or nil if not found.
func (m *Manager) Get(id int) *Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodes[id]
}

// Count returns the number of active nodes.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.nodes)
}

// List returns a snapshot of all active nodes.
func (m *Manager) List() []*Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, n := range m.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// NodeInfo holds summary information about a connected node.
type NodeInfo struct {
	ID       int
	UserName string
	Remote   string
	Menu     string
}

// ListInfo returns summary info for all active nodes.
func (m *Manager) ListInfo() []NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info := make([]NodeInfo, 0, len(m.nodes))
	for _, n := range m.nodes {
		name := n.UserName
		if name == "" {
			name = "(logging in)"
		}
		info = append(info, NodeInfo{
			ID:       n.ID,
			UserName: name,
			Remote:   n.Remote,
			Menu:     n.CurrentMenu,
		})
	}
	return info
}

// Broadcast sends a message to all connected nodes.
func (m *Manager) Broadcast(msg string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, n := range m.nodes {
		n.Term.SendLn(fmt.Sprintf("\r\n*** %s", msg))
	}
}

// SendTo sends a message to a specific node.
func (m *Manager) SendTo(nodeID int, msg string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d not found", nodeID)
	}
	return n.Term.SendLn(fmt.Sprintf("\r\n*** %s", msg))
}
