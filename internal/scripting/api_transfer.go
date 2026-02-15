package scripting

import (
	"io"
	"log"

	"github.com/notepid/twilight_bbs/internal/transfer"
	lua "github.com/yuin/gopher-lua"
)

// TransferAPI exposes file transfer functions (ZMODEM via SEXYZ) to Lua.
type TransferAPI struct {
	config       *transfer.Config
	binaryMode   func() (io.ReadWriter, func(), bool) // returns raw RW, cleanup, isTelnet
	nodeID       int
}

// NewTransferAPI creates a Lua transfer API.
func NewTransferAPI(config *transfer.Config, binaryMode func() (io.ReadWriter, func(), bool), nodeID int) *TransferAPI {
	return &TransferAPI{
		config:     config,
		binaryMode: binaryMode,
		nodeID:     nodeID,
	}
}

// Register installs transfer functions in the Lua state.
func (api *TransferAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("send", L.NewFunction(api.luaSend))
	mod.RawSetString("receive", L.NewFunction(api.luaReceive))
	mod.RawSetString("available", L.NewFunction(api.luaAvailable))

	L.SetGlobal("transfer", mod)
}

// luaSend handles: transfer.send(filepath | {filepath1, filepath2, ...} | filepath1, filepath2, ...)
// → (bool, errString|nil)
//
// Switches the connection to binary mode, runs SEXYZ sz (ZMODEM send),
// then restores normal mode.
func (api *TransferAPI) luaSend(L *lua.LState) int {
	argCount := L.GetTop()
	if argCount == 0 {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("missing file paths"))
		return 2
	}

	var filePaths []string
	if argCount == 1 {
		if tbl, ok := L.Get(1).(*lua.LTable); ok {
			tbl.ForEach(func(_ lua.LValue, v lua.LValue) {
				if s, ok := v.(lua.LString); ok {
					filePaths = append(filePaths, string(s))
				}
			})
		} else {
			filePaths = append(filePaths, L.CheckString(1))
		}
	} else {
		for i := 1; i <= argCount; i++ {
			filePaths = append(filePaths, L.CheckString(i))
		}
	}

	if len(filePaths) == 0 {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("no valid file paths"))
		return 2
	}

	if !api.config.Available() {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("SEXYZ binary not found"))
		return 2
	}

	rw, cleanup, isTelnet := api.binaryMode()
	if rw == nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("connection does not support binary mode"))
		return 2
	}
	defer cleanup()

	log.Printf("[transfer] Node %d: sending %d file(s)", api.nodeID, len(filePaths))

	result, err := api.config.Send(rw, isTelnet, filePaths...)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	_ = result
	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

// luaReceive handles: transfer.receive(uploadDir) → (table|nil, errString|nil)
//
// Switches the connection to binary mode, runs SEXYZ rz (ZMODEM receive),
// then restores normal mode. Returns a table of received files:
//
//	{ {name="file.zip", size=12345}, ... }
func (api *TransferAPI) luaReceive(L *lua.LState) int {
	uploadDir := L.CheckString(1)

	if !api.config.Available() {
		L.Push(lua.LNil)
		L.Push(lua.LString("SEXYZ binary not found"))
		return 2
	}

	rw, cleanup, isTelnet := api.binaryMode()
	if rw == nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("connection does not support binary mode"))
		return 2
	}
	defer cleanup()

	log.Printf("[transfer] Node %d: receiving files into %s", api.nodeID, uploadDir)

	result, err := api.config.Receive(rw, isTelnet, uploadDir)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Build a Lua table of received files.
	tbl := L.NewTable()
	for i, f := range result.Files {
		ft := L.NewTable()
		ft.RawSetString("name", lua.LString(f.Name))
		ft.RawSetString("size", lua.LNumber(f.Size))
		tbl.RawSetInt(i+1, ft)
	}

	L.Push(tbl)
	L.Push(lua.LNil)
	return 2
}

// luaAvailable handles: transfer.available() → bool
func (api *TransferAPI) luaAvailable(L *lua.LState) int {
	L.Push(lua.LBool(api.config.Available()))
	return 1
}
