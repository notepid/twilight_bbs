package scripting

import (
	"strings"

	"github.com/mikael/twilight_bbs/internal/terminal"
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

	// Inter-node callbacks (Phase 7+)
	OnShowOnline func() error
	OnEnterChat  func() error
	OnLaunchDoor func(name string) error
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

	// Methods - Navigation
	case "goto_menu":
		L.Push(L.NewFunction(api.luaGotoMenu))
	case "gosub_menu":
		L.Push(L.NewFunction(api.luaGosubMenu))
	case "return_menu":
		L.Push(L.NewFunction(api.luaReturnMenu))
	case "disconnect":
		L.Push(L.NewFunction(api.luaDisconnect))

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
