package scripting

import (
	"fmt"
	"log"

	lua "github.com/yuin/gopher-lua"
)

// VM wraps a Lua state with BBS-specific configuration.
type VM struct {
	L *lua.LState
}

// NewVM creates a new Lua VM with the standard libraries loaded.
func NewVM() *VM {
	L := lua.NewState(lua.Options{
		CallStackSize: 120,
		RegistrySize:  120 * 20,
	})

	return &VM{L: L}
}

// Close shuts down the Lua VM.
func (vm *VM) Close() {
	vm.L.Close()
}

// LoadScript loads and executes a Lua script file.
// The script is expected to return a table with menu handler functions.
func (vm *VM) LoadScript(path string) error {
	if err := vm.L.DoFile(path); err != nil {
		return fmt.Errorf("load script %s: %w", path, err)
	}
	return nil
}

// CallMenuHandler calls a function on the menu table returned by the script.
// The menu table should be at the top of the stack after DoFile.
func (vm *VM) CallMenuHandler(funcName string, args ...lua.LValue) error {
	// Get the menu table from the return value or global
	menuTable := vm.getMenuTable()
	if menuTable == nil {
		return fmt.Errorf("no menu table found")
	}

	fn := menuTable.RawGetString(funcName)
	if fn == lua.LNil {
		// Function not defined - not an error, just skip
		return nil
	}

	if _, ok := fn.(*lua.LFunction); !ok {
		return fmt.Errorf("menu.%s is not a function", funcName)
	}

	if err := vm.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, args...); err != nil {
		return fmt.Errorf("call menu.%s: %w", funcName, err)
	}

	return nil
}

// HasMenuHandler checks if the menu table has a specific handler function.
func (vm *VM) HasMenuHandler(funcName string) bool {
	menuTable := vm.getMenuTable()
	if menuTable == nil {
		return false
	}
	fn := menuTable.RawGetString(funcName)
	_, ok := fn.(*lua.LFunction)
	return ok
}

// getMenuTable finds the menu table - either as the return value of the
// script or as a global named "menu".
func (vm *VM) getMenuTable() *lua.LTable {
	// Check top of stack first (return value)
	top := vm.L.Get(-1)
	if tbl, ok := top.(*lua.LTable); ok {
		return tbl
	}

	// Check global "menu"
	g := vm.L.GetGlobal("menu")
	if tbl, ok := g.(*lua.LTable); ok {
		return tbl
	}

	return nil
}

// SetGlobal sets a global value in the Lua state.
func (vm *VM) SetGlobal(name string, value lua.LValue) {
	vm.L.SetGlobal(name, value)
}

// RegisterModule registers a table of functions as a Lua module.
func (vm *VM) RegisterModule(name string, funcs map[string]lua.LGFunction) {
	mod := vm.L.NewTable()
	for fname, fn := range funcs {
		mod.RawSetString(fname, vm.L.NewFunction(fn))
	}
	vm.L.SetGlobal(name, mod)
}

// LogError logs a Lua error with context.
func LogError(context string, err error) {
	if err != nil {
		log.Printf("Lua error [%s]: %v", context, err)
	}
}
