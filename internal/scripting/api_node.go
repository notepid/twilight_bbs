package scripting

import (
	"strings"

	"github.com/notepid/twilight_bbs/internal/ansi"
	"github.com/notepid/twilight_bbs/internal/terminal"
	lua "github.com/yuin/gopher-lua"
)

// NodeAPI exposes BBS node I/O and navigation functions to Lua.
type NodeAPI struct {
	term *terminal.Terminal

	// Navigation callbacks - set by the menu engine
	OnGotoMenu   func(name string) error
	OnGosubMenu  func(name string) error
	OnReturnMenu func() error
	OnDisconnect func()
	OnDisplay    func(name string) error

	// State callbacks - set by the menu engine
	OnSetMenuState func(menuName, key string, value interface{})
	OnGetMenuState func(menuName, key string) (interface{}, bool)
	OnGetField     func(id string) (ansi.Field, bool)

	// Inter-node callbacks (Phase 7+)
	OnShowOnline func() error
	OnEnterChat  func() error
	OnLaunchDoor func(name string) error

	// Current menu name for state access
	CurrentMenuName string
}

// NewNodeAPI creates a Lua API instance bound to a terminal.
func NewNodeAPI(term *terminal.Terminal) *NodeAPI {
	return &NodeAPI{term: term}
}

// Register installs the node userdata type and creates a node instance
// in the Lua state, returning it as a LValue for use as function arguments.
func (api *NodeAPI) Register(L *lua.LState) *lua.LUserData {
	// Register the "node" type
	mt := L.NewTypeMetatable("node")
	L.SetField(mt, "__index", L.NewFunction(api.nodeIndex))

	// Create the node userdata
	ud := L.NewUserData()
	ud.Value = api
	L.SetMetatable(ud, L.GetTypeMetatable("node"))

	// Also set as global "node" for convenience
	L.SetGlobal("node", ud)

	return ud
}

// nodeIndex is the __index metamethod for the node userdata.
// It provides both methods (functions) and properties.
func (api *NodeAPI) nodeIndex(L *lua.LState) int {
	key := L.CheckString(2)

	switch key {
	// Methods - I/O
	case "send":
		L.Push(L.NewFunction(api.luaSend))
	case "sendln":
		L.Push(L.NewFunction(api.luaSendLn))
	case "cls":
		L.Push(L.NewFunction(api.luaCls))
	case "display":
		L.Push(L.NewFunction(api.luaDisplay))
	case "goto_xy":
		L.Push(L.NewFunction(api.luaGotoXY))
	case "color":
		L.Push(L.NewFunction(api.luaColor))
	case "pause":
		L.Push(L.NewFunction(api.luaPause))
	case "more":
		L.Push(L.NewFunction(api.luaPause)) // alias
	case "save_cursor":
		L.Push(L.NewFunction(api.luaSaveCursor))
	case "restore_cursor":
		L.Push(L.NewFunction(api.luaRestoreCursor))
	case "cursor_off":
		L.Push(L.NewFunction(api.luaCursorOff))
	case "cursor_on":
		L.Push(L.NewFunction(api.luaCursorOn))

	// Methods - Input
	case "getkey":
		L.Push(L.NewFunction(api.luaGetKey))
	case "getline":
		L.Push(L.NewFunction(api.luaGetLine))
	case "hotkey":
		L.Push(L.NewFunction(api.luaHotkey))
	case "ask":
		L.Push(L.NewFunction(api.luaAsk))
	case "password":
		L.Push(L.NewFunction(api.luaPassword))
	case "yesno":
		L.Push(L.NewFunction(api.luaYesNo))

	// Methods - Fields
	case "field":
		L.Push(L.NewFunction(api.luaField))
	case "edit_field":
		L.Push(L.NewFunction(api.luaEditField))
	case "input_field":
		L.Push(L.NewFunction(api.luaInputField))
	case "password_field":
		L.Push(L.NewFunction(api.luaPasswordField))

	// Methods - Navigation
	case "goto_menu":
		L.Push(L.NewFunction(api.luaGotoMenu))
	case "gosub_menu":
		L.Push(L.NewFunction(api.luaGosubMenu))
	case "return_menu":
		L.Push(L.NewFunction(api.luaReturnMenu))
	case "disconnect":
		L.Push(L.NewFunction(api.luaDisconnect))

	// Methods - State
	case "set_state":
		L.Push(L.NewFunction(api.luaSetState))
	case "get_state":
		L.Push(L.NewFunction(api.luaGetState))

	// Methods - Inter-node (Phase 7+)
	case "show_online":
		L.Push(L.NewFunction(api.luaShowOnline))
	case "enter_chat":
		L.Push(L.NewFunction(api.luaEnterChat))
	case "launch_door":
		L.Push(L.NewFunction(api.luaLaunchDoor))

	// Properties
	case "width":
		L.Push(lua.LNumber(api.term.Width))
	case "height":
		L.Push(lua.LNumber(api.term.Height))
	case "ansi":
		L.Push(lua.LBool(api.term.ANSIEnabled))

	default:
		L.Push(lua.LNil)
	}

	return 1
}

// --- I/O Methods ---

func (api *NodeAPI) luaSend(L *lua.LState) int {
	text := L.CheckString(2)
	api.term.Send(text)
	return 0
}

func (api *NodeAPI) luaSendLn(L *lua.LState) int {
	text := L.CheckString(2)
	api.term.SendLn(text)
	return 0
}

func (api *NodeAPI) luaCls(L *lua.LState) int {
	api.term.Cls()
	return 0
}

func (api *NodeAPI) luaDisplay(L *lua.LState) int {
	name := strings.TrimSpace(L.CheckString(2))
	if name == "" {
		L.ArgError(2, "empty display name")
		return 0
	}
	if api.OnDisplay != nil {
		if err := api.OnDisplay(name); err != nil {
			L.ArgError(2, err.Error())
		}
	}
	return 0
}

func (api *NodeAPI) luaGotoXY(L *lua.LState) int {
	row := L.CheckInt(2)
	col := L.CheckInt(3)
	api.term.GotoXY(row, col)
	return 0
}

func (api *NodeAPI) luaColor(L *lua.LState) int {
	fg := L.CheckInt(2)
	bg := L.OptInt(3, -1)
	api.term.SetColor(fg, bg)
	return 0
}

func (api *NodeAPI) luaPause(L *lua.LState) int {
	api.term.Pause()
	return 0
}

func (api *NodeAPI) luaSaveCursor(L *lua.LState) int {
	if api.term.ANSIEnabled {
		api.term.Send(terminal.SaveCursor())
	}
	return 0
}

func (api *NodeAPI) luaRestoreCursor(L *lua.LState) int {
	if api.term.ANSIEnabled {
		api.term.Send(terminal.RestoreCursor())
	}
	return 0
}

func (api *NodeAPI) luaCursorOff(L *lua.LState) int {
	if api.term.ANSIEnabled {
		api.term.Send(terminal.HideCursor())
	}
	return 0
}

func (api *NodeAPI) luaCursorOn(L *lua.LState) int {
	if api.term.ANSIEnabled {
		api.term.Send(terminal.ShowCursor())
	}
	return 0
}

// --- Input Methods ---

func (api *NodeAPI) luaGetKey(L *lua.LState) int {
	key, err := api.term.GetKey()
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(string(key)))
	return 1
}

func (api *NodeAPI) luaGetLine(L *lua.LState) int {
	maxLen := L.OptInt(2, 80)
	line, err := api.term.GetLine(maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(line))
	return 1
}

func (api *NodeAPI) luaHotkey(L *lua.LState) int {
	prompt := L.CheckString(2)
	key, err := api.term.Hotkey(prompt)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(string(key)))
	return 1
}

func (api *NodeAPI) luaAsk(L *lua.LState) int {
	prompt := L.CheckString(2)
	maxLen := L.OptInt(3, 80)
	line, err := api.term.Ask(prompt, maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(line))
	return 1
}

func (api *NodeAPI) luaPassword(L *lua.LState) int {
	maxLen := L.OptInt(2, 40)
	pass, err := api.term.GetPassword(maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(pass))
	return 1
}

func (api *NodeAPI) luaYesNo(L *lua.LState) int {
	prompt := L.CheckString(2)
	result, err := api.term.YesNo(prompt)
	if err != nil {
		L.Push(lua.LFalse)
		return 1
	}
	L.Push(lua.LBool(result))
	return 1
}

// --- Field Methods ---

func (api *NodeAPI) luaField(L *lua.LState) int {
	id := strings.TrimSpace(L.CheckString(2))
	if id == "" || api.OnGetField == nil {
		L.Push(lua.LNil)
		return 1
	}

	f, ok := api.OnGetField(id)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	L.Push(lua.LNumber(f.Row))
	L.Push(lua.LNumber(f.Col))
	L.Push(lua.LNumber(f.MaxLen))
	return 3
}

func (api *NodeAPI) luaEditField(L *lua.LState) int {
	id := strings.TrimSpace(L.CheckString(2))
	if id == "" || api.OnGetField == nil {
		L.Push(lua.LNil)
		return 1
	}

	f, ok := api.OnGetField(id)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	override := L.OptInt(3, 0)
	maxLen := override
	if maxLen <= 0 {
		maxLen = f.MaxLen
	}
	if maxLen <= 0 {
		maxLen = 80
	}

	api.term.GotoXY(f.Row, f.Col)
	line, err := api.term.GetLine(maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(line))
	return 1
}

func (api *NodeAPI) luaInputField(L *lua.LState) int {
	id := strings.TrimSpace(L.CheckString(2))
	if id == "" || api.OnGetField == nil {
		L.Push(lua.LNil)
		return 1
	}

	f, ok := api.OnGetField(id)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	override := L.OptInt(3, 0)
	maxLen := override
	if maxLen <= 0 {
		maxLen = f.MaxLen
	}
	if maxLen <= 0 {
		maxLen = 80
	}

	api.term.GotoXY(f.Row, f.Col)
	api.term.Send(strings.Repeat(" ", maxLen))
	api.term.GotoXY(f.Row, f.Col)

	line, err := api.term.GetLine(maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(line))
	return 1
}

func (api *NodeAPI) luaPasswordField(L *lua.LState) int {
	id := strings.TrimSpace(L.CheckString(2))
	if id == "" || api.OnGetField == nil {
		L.Push(lua.LNil)
		return 1
	}

	f, ok := api.OnGetField(id)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	override := L.OptInt(3, 0)
	maxLen := override
	if maxLen <= 0 {
		maxLen = f.MaxLen
	}
	if maxLen <= 0 {
		maxLen = 40
	}

	api.term.GotoXY(f.Row, f.Col)
	api.term.Send(strings.Repeat(" ", maxLen))
	api.term.GotoXY(f.Row, f.Col)

	pass, err := api.term.GetPassword(maxLen)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(pass))
	return 1
}

// --- Navigation Methods ---

func (api *NodeAPI) luaGotoMenu(L *lua.LState) int {
	name := L.CheckString(2)
	if api.OnGotoMenu != nil {
		if err := api.OnGotoMenu(name); err != nil {
			L.RaiseError("goto_menu %s: %s", name, err.Error())
		}
	}
	return 0
}

func (api *NodeAPI) luaGosubMenu(L *lua.LState) int {
	name := L.CheckString(2)
	if api.OnGosubMenu != nil {
		if err := api.OnGosubMenu(name); err != nil {
			L.RaiseError("gosub_menu %s: %s", name, err.Error())
		}
	}
	return 0
}

func (api *NodeAPI) luaReturnMenu(L *lua.LState) int {
	if api.OnReturnMenu != nil {
		if err := api.OnReturnMenu(); err != nil {
			L.RaiseError("return_menu: %s", err.Error())
		}
	}
	return 0
}

func (api *NodeAPI) luaDisconnect(L *lua.LState) int {
	if api.OnDisconnect != nil {
		api.OnDisconnect()
	}
	return 0
}

// --- State Methods ---

func (api *NodeAPI) luaSetState(L *lua.LState) int {
	key := L.CheckString(2)
	value := L.CheckAny(3)

	if api.OnSetMenuState == nil {
		return 0
	}

	// Convert Lua value to Go interface{}
	var val interface{}
	switch v := value.(type) {
	case lua.LBool:
		val = bool(v)
	case lua.LNumber:
		val = float64(v)
	case lua.LString:
		val = string(v)
	default:
		// For nil and other types, store as nil
		if value == lua.LNil {
			val = nil
		} else {
			val = nil
		}
	}

	api.OnSetMenuState(api.CurrentMenuName, key, val)
	return 0
}

func (api *NodeAPI) luaGetState(L *lua.LState) int {
	key := L.CheckString(2)

	if api.OnGetMenuState == nil {
		L.Push(lua.LNil)
		return 1
	}

	val, ok := api.OnGetMenuState(api.CurrentMenuName, key)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}

	// Convert Go interface{} to Lua value
	switch v := val.(type) {
	case nil:
		L.Push(lua.LNil)
	case bool:
		L.Push(lua.LBool(v))
	case float64:
		L.Push(lua.LNumber(v))
	case string:
		L.Push(lua.LString(v))
	case int:
		L.Push(lua.LNumber(lua.LNumber(v)))
	default:
		L.Push(lua.LNil)
	}

	return 1
}

// --- Inter-node Methods (stubs, implemented in later phases) ---

func (api *NodeAPI) luaShowOnline(L *lua.LState) int {
	if api.OnShowOnline != nil {
		api.OnShowOnline()
	}
	return 0
}

func (api *NodeAPI) luaEnterChat(L *lua.LState) int {
	if api.OnEnterChat != nil {
		api.OnEnterChat()
	}
	return 0
}

func (api *NodeAPI) luaLaunchDoor(L *lua.LState) int {
	name := L.CheckString(2)
	if api.OnLaunchDoor != nil {
		api.OnLaunchDoor(name)
	}
	return 0
}
