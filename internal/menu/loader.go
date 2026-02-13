package menu

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Registry holds all discovered menus and provides lookup.
type Registry struct {
	mu    sync.RWMutex
	menus map[string]*Menu
	dirs  []string
}

// NewRegistry creates a new menu registry that scans the given directories.
func NewRegistry(dirs ...string) *Registry {
	return &Registry{
		menus: make(map[string]*Menu),
		dirs:  dirs,
	}
}

// Scan discovers all menu files in the configured directories.
// Files sharing a base name (e.g., main_menu.ans, main_menu.asc, main_menu.lua)
// are grouped into a single Menu entry.
func (r *Registry) Scan() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.menus = make(map[string]*Menu)

	for _, dir := range r.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Menu directory does not exist: %s", dir)
				continue
			}
			return fmt.Errorf("scan menu dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			ext := strings.ToLower(filepath.Ext(name))
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			fullPath := filepath.Join(dir, name)

			// Get or create menu entry
			m, ok := r.menus[baseName]
			if !ok {
				m = &Menu{Name: baseName}
				r.menus[baseName] = m
			}

			switch ext {
			case ".ans":
				m.ANSPath = fullPath
			case ".asc":
				m.ASCPath = fullPath
			case ".lua":
				m.ScriptPath = fullPath
			}
		}
	}

	log.Printf("Loaded %d menus", len(r.menus))
	for name, m := range r.menus {
		parts := []string{}
		if m.HasANS() {
			parts = append(parts, "ANS")
		}
		if m.HasASC() {
			parts = append(parts, "ASC")
		}
		if m.HasScript() {
			parts = append(parts, "LUA")
		}
		log.Printf("  Menu: %s [%s]", name, strings.Join(parts, "+"))
	}

	return nil
}

// Get returns a menu by name, or nil if not found.
func (r *Registry) Get(name string) *Menu {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.menus[name]
}

// List returns the names of all registered menus.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.menus))
	for name := range r.menus {
		names = append(names, name)
	}
	return names
}

// Reload rescans the menu directories.
func (r *Registry) Reload() error {
	return r.Scan()
}
