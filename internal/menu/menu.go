package menu

// Menu represents a loaded BBS menu with its display file and script.
type Menu struct {
	Name       string // base name (e.g., "main_menu")
	ANSPath    string // path to .ans file (may be empty)
	ASCPath    string // path to .asc file (may be empty)
	ScriptPath string // path to .lua file (may be empty)
}

// HasANS returns true if an ANSI display file exists for this menu.
func (m *Menu) HasANS() bool {
	return m.ANSPath != ""
}

// HasASC returns true if an ASCII display file exists for this menu.
func (m *Menu) HasASC() bool {
	return m.ASCPath != ""
}

// HasScript returns true if a Lua script exists for this menu.
func (m *Menu) HasScript() bool {
	return m.ScriptPath != ""
}

// DisplayPath returns the appropriate display file path based on ANSI capability.
func (m *Menu) DisplayPath(ansiEnabled bool) string {
	if ansiEnabled && m.HasANS() {
		return m.ANSPath
	}
	if m.HasASC() {
		return m.ASCPath
	}
	// Fall back to ANS even if ANSI is "disabled" - better than nothing
	return m.ANSPath
}
