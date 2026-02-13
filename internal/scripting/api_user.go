package scripting

import (
	"github.com/notepid/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// UserAPI exposes user-related functions to Lua.
type UserAPI struct {
	repo        *user.Repo
	currentUser *user.User

	// Callback when user logs in
	OnLogin func(u *user.User)
}

// NewUserAPI creates a Lua user API.
func NewUserAPI(repo *user.Repo) *UserAPI {
	return &UserAPI{repo: repo}
}

// SetCurrentUser updates the current user reference.
func (api *UserAPI) SetCurrentUser(u *user.User) {
	api.currentUser = u
}

// Register installs user functions in the Lua state.
func (api *UserAPI) Register(L *lua.LState) {
	userMod := L.NewTable()

	userMod.RawSetString("login", L.NewFunction(api.luaLogin))
	userMod.RawSetString("register", L.NewFunction(api.luaRegister))
	userMod.RawSetString("exists", L.NewFunction(api.luaExists))
	userMod.RawSetString("get_current", L.NewFunction(api.luaGetCurrent))
	userMod.RawSetString("update_profile", L.NewFunction(api.luaUpdateProfile))
	userMod.RawSetString("update_password", L.NewFunction(api.luaUpdatePassword))
	userMod.RawSetString("list", L.NewFunction(api.luaList))

	L.SetGlobal("users", userMod)
}

func (api *UserAPI) luaLogin(L *lua.LState) int {
	username := L.CheckString(1)
	password := L.CheckString(2)

	u, err := api.repo.Authenticate(username, password)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	api.currentUser = u
	if api.OnLogin != nil {
		api.OnLogin(u)
	}

	L.Push(api.userToTable(L, u))
	L.Push(lua.LNil)
	return 2
}

func (api *UserAPI) luaRegister(L *lua.LState) int {
	username := L.CheckString(1)
	password := L.CheckString(2)
	realName := L.OptString(3, "")
	location := L.OptString(4, "")
	email := L.OptString(5, "")

	if api.repo.Exists(username) {
		L.Push(lua.LNil)
		L.Push(lua.LString("username already exists"))
		return 2
	}

	u, err := api.repo.Create(username, password, realName, location, email)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	api.currentUser = u
	if api.OnLogin != nil {
		api.OnLogin(u)
	}

	L.Push(api.userToTable(L, u))
	L.Push(lua.LNil)
	return 2
}

func (api *UserAPI) luaExists(L *lua.LState) int {
	username := L.CheckString(1)
	L.Push(lua.LBool(api.repo.Exists(username)))
	return 1
}

func (api *UserAPI) luaGetCurrent(L *lua.LState) int {
	if api.currentUser == nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(api.userToTable(L, api.currentUser))
	return 1
}

func (api *UserAPI) luaUpdateProfile(L *lua.LState) int {
	if api.currentUser == nil {
		L.Push(lua.LString("not logged in"))
		return 1
	}
	realName := L.CheckString(1)
	location := L.CheckString(2)
	email := L.CheckString(3)

	if err := api.repo.UpdateProfile(api.currentUser.ID, realName, location, email); err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}

	api.currentUser.RealName = realName
	api.currentUser.Location = location
	api.currentUser.Email = email
	L.Push(lua.LNil)
	return 1
}

func (api *UserAPI) luaUpdatePassword(L *lua.LState) int {
	if api.currentUser == nil {
		L.Push(lua.LString("not logged in"))
		return 1
	}
	newPassword := L.CheckString(1)

	if err := api.repo.UpdatePassword(api.currentUser.ID, newPassword); err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func (api *UserAPI) luaList(L *lua.LState) int {
	users, err := api.repo.List()
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, u := range users {
		tbl.RawSetInt(i+1, api.userToTable(L, u))
	}
	L.Push(tbl)
	return 1
}

// userToTable converts a User struct to a Lua table.
func (api *UserAPI) userToTable(L *lua.LState, u *user.User) *lua.LTable {
	tbl := L.NewTable()
	tbl.RawSetString("id", lua.LNumber(u.ID))
	tbl.RawSetString("name", lua.LString(u.Username))
	tbl.RawSetString("real_name", lua.LString(u.RealName))
	tbl.RawSetString("location", lua.LString(u.Location))
	tbl.RawSetString("email", lua.LString(u.Email))
	tbl.RawSetString("level", lua.LNumber(u.SecurityLevel))
	tbl.RawSetString("calls", lua.LNumber(u.TotalCalls))
	tbl.RawSetString("ansi", lua.LBool(u.ANSIEnabled))
	if u.LastCallAt != nil {
		tbl.RawSetString("last_on", lua.LString(u.LastCallAt.Format("2006-01-02 15:04")))
	}
	tbl.RawSetString("created", lua.LString(u.CreatedAt.Format("2006-01-02")))
	return tbl
}
