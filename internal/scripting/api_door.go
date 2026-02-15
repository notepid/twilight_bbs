package scripting

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/notepid/twilight_bbs/internal/door"
	"github.com/notepid/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// DoorAPI exposes door launching functions to Lua.
type DoorAPI struct {
	launcher    *door.Launcher
	currentUser func() *user.User
	termSize    func() (width, height int)
	nodeID      int
	stdin       io.Reader
	stdout      io.Writer
}

// NewDoorAPI creates a Lua door API.
func NewDoorAPI(launcher *door.Launcher, currentUser func() *user.User, termSize func() (int, int), nodeID int, stdin io.Reader, stdout io.Writer) *DoorAPI {
	return &DoorAPI{
		launcher:    launcher,
		currentUser: currentUser,
		termSize:    termSize,
		nodeID:      nodeID,
		stdin:       stdin,
		stdout:      stdout,
	}
}

// Register installs door functions in the Lua state.
func (api *DoorAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("launch", L.NewFunction(api.luaLaunch))
	mod.RawSetString("available", L.NewFunction(api.luaAvailable))

	L.SetGlobal("door", mod)
}

func (api *DoorAPI) luaLaunch(L *lua.LState) int {
	u := api.currentUser()
	if u == nil {
		L.Push(lua.LString("not logged in"))
		return 1
	}

	if !api.launcher.Available() {
		L.Push(lua.LString("dosemu2 is not installed"))
		return 1
	}

	// door.launch(cfgTable)
	cfgTable := L.CheckTable(1)
	cfg, err := parseDoorConfigFromLua(cfgTable)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}

	if u.SecurityLevel < cfg.SecurityLevel {
		L.Push(lua.LString("insufficient security level"))
		return 1
	}

	termW, termH := api.termSize()
	session := &door.Session{
		DoorConfig:   &cfg,
		User:         u,
		NodeID:       api.nodeID,
		TimeLeftMins: 60,
		ComPort:      1,
		BaudRate:     115200,
		DosemuPath:   api.launcher.DosemuPath,
		DriveCPath:   api.launcher.DriveCPath,
		TermWidth:    termW,
		TermHeight:   termH,
	}

	log.Printf("Node %d launching door: %s", api.nodeID, cfg.Name)

	if err := api.launcher.Launch(session, api.stdin, api.stdout); err != nil {
		L.Push(lua.LString(fmt.Sprintf("door error: %v", err)))
		return 1
	}

	L.Push(lua.LNil)
	return 1
}

func (api *DoorAPI) luaAvailable(L *lua.LState) int {
	L.Push(lua.LBool(api.launcher.Available()))
	return 1
}

func parseDoorConfigFromLua(t *lua.LTable) (door.Config, error) {
	// Supported fields:
	// - name (string, required)
	// - description (string, optional)
	// - command (string, required)
	// - drop_file_type (string, optional; default DOOR.SYS)
	// - security_level (number, optional; default 10)
	// - multiuser (bool, optional; default true)
	getString := func(key string) string {
		v := t.RawGetString(key)
		if s, ok := v.(lua.LString); ok {
			return string(s)
		}
		return ""
	}
	getBool := func(key string) (bool, bool) {
		v := t.RawGetString(key)
		if b, ok := v.(lua.LBool); ok {
			return bool(b), true
		}
		return false, false
	}
	getNumber := func(key string) (float64, bool) {
		v := t.RawGetString(key)
		if n, ok := v.(lua.LNumber); ok {
			return float64(n), true
		}
		return 0, false
	}

	name := strings.TrimSpace(getString("name"))
	if name == "" {
		return door.Config{}, fmt.Errorf("door config missing 'name'")
	}

	command := strings.TrimSpace(getString("command"))
	if command == "" {
		return door.Config{}, fmt.Errorf("door '%s' missing 'command'", name)
	}

	desc := strings.TrimSpace(getString("description"))

	drop := strings.TrimSpace(getString("drop_file_type"))
	if drop == "" {
		drop = "DOOR.SYS"
	}

	secLevel := 10
	if n, ok := getNumber("security_level"); ok {
		if n < 0 {
			secLevel = 0
		} else {
			secLevel = int(n)
		}
	}

	multiUser := true
	if b, ok := getBool("multiuser"); ok {
		multiUser = b
	} else if b, ok := getBool("multi_user"); ok {
		multiUser = b
	}

	return door.Config{
		ID:            0,
		Name:          name,
		Description:   desc,
		Command:       command,
		DropFileType:  drop,
		SecurityLevel: secLevel,
		MultiUser:     multiUser,
	}, nil
}
