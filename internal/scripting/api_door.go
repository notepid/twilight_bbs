package scripting

import (
	"database/sql"
	"fmt"
	"io"
	"log"

	"github.com/notepid/twilight_bbs/internal/door"
	"github.com/notepid/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// DoorAPI exposes door launching functions to Lua.
type DoorAPI struct {
	db          *sql.DB
	launcher    *door.Launcher
	currentUser func() *user.User
	nodeID      int
	stdin       io.Reader
	stdout      io.Writer
}

// NewDoorAPI creates a Lua door API.
func NewDoorAPI(db *sql.DB, launcher *door.Launcher, currentUser func() *user.User,
	nodeID int, stdin io.Reader, stdout io.Writer) *DoorAPI {
	return &DoorAPI{
		db:          db,
		launcher:    launcher,
		currentUser: currentUser,
		nodeID:      nodeID,
		stdin:       stdin,
		stdout:      stdout,
	}
}

// Register installs door functions in the Lua state.
func (api *DoorAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("list", L.NewFunction(api.luaList))
	mod.RawSetString("launch", L.NewFunction(api.luaLaunch))
	mod.RawSetString("available", L.NewFunction(api.luaAvailable))

	L.SetGlobal("door", mod)
}

func (api *DoorAPI) luaList(L *lua.LState) int {
	u := api.currentUser()
	level := 0
	if u != nil {
		level = u.SecurityLevel
	}

	rows, err := api.db.Query(`
		SELECT id, name, description, security_level
		FROM doors WHERE security_level <= ?
		ORDER BY name
	`, level)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	defer rows.Close()

	tbl := L.NewTable()
	i := 1
	for rows.Next() {
		var id, secLevel int
		var name, desc string
		if err := rows.Scan(&id, &name, &desc, &secLevel); err != nil {
			continue
		}
		dt := L.NewTable()
		dt.RawSetString("id", lua.LNumber(id))
		dt.RawSetString("name", lua.LString(name))
		dt.RawSetString("description", lua.LString(desc))
		tbl.RawSetInt(i, dt)
		i++
	}
	L.Push(tbl)
	return 1
}

func (api *DoorAPI) luaLaunch(L *lua.LState) int {
	name := L.CheckString(1)

	u := api.currentUser()
	if u == nil {
		L.Push(lua.LString("not logged in"))
		return 1
	}

	// Look up door config
	var cfg door.Config
	err := api.db.QueryRow(`
		SELECT id, name, description, command, drop_file_type, security_level
		FROM doors WHERE name = ?
	`, name).Scan(&cfg.ID, &cfg.Name, &cfg.Description, &cfg.Command,
		&cfg.DropFileType, &cfg.SecurityLevel)
	if err != nil {
		L.Push(lua.LString(fmt.Sprintf("door '%s' not found", name)))
		return 1
	}

	if u.SecurityLevel < cfg.SecurityLevel {
		L.Push(lua.LString("insufficient security level"))
		return 1
	}

	if !api.launcher.Available() {
		L.Push(lua.LString("dosemu2 is not installed"))
		return 1
	}

	session := &door.Session{
		DoorConfig:   &cfg,
		User:         u,
		NodeID:       api.nodeID,
		TimeLeftMins: 60,
		ComPort:      0,
		BaudRate:     115200,
		DosemuPath:   api.launcher.DosemuPath,
		DriveCPath:   api.launcher.DriveCPath,
	}

	log.Printf("Node %d launching door: %s", api.nodeID, name)

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
