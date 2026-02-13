package scripting

import (
	"github.com/mikael/twilight_bbs/internal/message"
	"github.com/mikael/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// MessageAPI exposes message base functions to Lua.
type MessageAPI struct {
	repo        *message.Repo
	currentUser func() *user.User
}

// NewMessageAPI creates a Lua message API.
func NewMessageAPI(repo *message.Repo, currentUser func() *user.User) *MessageAPI {
	return &MessageAPI{repo: repo, currentUser: currentUser}
}

// Register installs message functions in the Lua state.
func (api *MessageAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("areas", L.NewFunction(api.luaAreas))
	mod.RawSetString("get_area", L.NewFunction(api.luaGetArea))
	mod.RawSetString("list", L.NewFunction(api.luaList))
	mod.RawSetString("read", L.NewFunction(api.luaRead))
	mod.RawSetString("post", L.NewFunction(api.luaPost))
	mod.RawSetString("scan_new", L.NewFunction(api.luaScanNew))
	mod.RawSetString("mark_read", L.NewFunction(api.luaMarkRead))
	mod.RawSetString("count", L.NewFunction(api.luaCount))

	L.SetGlobal("msg", mod)
}

func (api *MessageAPI) luaAreas(L *lua.LState) int {
	u := api.currentUser()
	level := 0
	userID := 0
	if u != nil {
		level = u.SecurityLevel
		userID = u.ID
	}

	areas, err := api.repo.ListAreasWithNew(userID, level)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, a := range areas {
		at := L.NewTable()
		at.RawSetString("id", lua.LNumber(a.ID))
		at.RawSetString("name", lua.LString(a.Name))
		at.RawSetString("description", lua.LString(a.Description))
		at.RawSetString("total", lua.LNumber(a.TotalMsgs))
		at.RawSetString("new", lua.LNumber(a.NewMsgs))
		at.RawSetString("read_level", lua.LNumber(a.ReadLevel))
		at.RawSetString("write_level", lua.LNumber(a.WriteLevel))
		tbl.RawSetInt(i+1, at)
	}
	L.Push(tbl)
	return 1
}

func (api *MessageAPI) luaGetArea(L *lua.LState) int {
	areaID := L.CheckInt(1)
	a, err := api.repo.GetArea(areaID)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	at := L.NewTable()
	at.RawSetString("id", lua.LNumber(a.ID))
	at.RawSetString("name", lua.LString(a.Name))
	at.RawSetString("description", lua.LString(a.Description))
	at.RawSetString("total", lua.LNumber(api.repo.CountMessages(a.ID)))
	L.Push(at)
	return 1
}

func (api *MessageAPI) luaList(L *lua.LState) int {
	areaID := L.CheckInt(1)
	offset := L.OptInt(2, 0)
	limit := L.OptInt(3, 20)

	messages, err := api.repo.ListMessages(areaID, offset, limit)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, m := range messages {
		mt := api.msgToTable(L, m, false)
		tbl.RawSetInt(i+1, mt)
	}
	L.Push(tbl)
	return 1
}

func (api *MessageAPI) luaRead(L *lua.LState) int {
	msgID := L.CheckInt(1)

	m, err := api.repo.GetMessage(msgID)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	// Auto-mark as read
	u := api.currentUser()
	if u != nil {
		api.repo.MarkRead(u.ID, m.AreaID, m.ID)
	}

	L.Push(api.msgToTable(L, m, true))
	return 1
}

func (api *MessageAPI) luaPost(L *lua.LState) int {
	u := api.currentUser()
	if u == nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("not logged in"))
		return 2
	}

	areaID := L.CheckInt(1)
	subject := L.CheckString(2)
	body := L.CheckString(3)
	toStr := L.OptString(4, "")

	var toUserID *int
	if toStr != "" {
		// Look up recipient - we'd need user repo here
		// For now, post as public
	}

	var replyToID *int
	replyTo := L.OptInt(5, 0)
	if replyTo > 0 {
		replyToID = &replyTo
	}

	id, err := api.repo.Post(areaID, u.ID, toUserID, subject, body, replyToID)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LNumber(id))
	L.Push(lua.LNil)
	return 2
}

func (api *MessageAPI) luaScanNew(L *lua.LState) int {
	u := api.currentUser()
	if u == nil {
		L.Push(lua.LNil)
		return 1
	}

	areaID := L.CheckInt(1)
	messages, err := api.repo.GetNewMessages(u.ID, areaID)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, m := range messages {
		mt := api.msgToTable(L, m, true)
		tbl.RawSetInt(i+1, mt)
	}
	L.Push(tbl)
	return 1
}

func (api *MessageAPI) luaMarkRead(L *lua.LState) int {
	u := api.currentUser()
	if u == nil {
		return 0
	}
	areaID := L.CheckInt(1)
	msgID := L.CheckInt(2)
	api.repo.MarkRead(u.ID, areaID, msgID)
	return 0
}

func (api *MessageAPI) luaCount(L *lua.LState) int {
	areaID := L.CheckInt(1)
	L.Push(lua.LNumber(api.repo.CountMessages(areaID)))
	return 1
}

// msgToTable converts a Message to a Lua table.
func (api *MessageAPI) msgToTable(L *lua.LState, m *message.Message, includeBody bool) *lua.LTable {
	mt := L.NewTable()
	mt.RawSetString("id", lua.LNumber(m.ID))
	mt.RawSetString("area_id", lua.LNumber(m.AreaID))
	mt.RawSetString("from", lua.LString(m.FromName))
	mt.RawSetString("from_id", lua.LNumber(m.FromUserID))
	mt.RawSetString("to", lua.LString(m.ToName))
	mt.RawSetString("subject", lua.LString(m.Subject))
	mt.RawSetString("date", lua.LString(m.CreatedAt.Format("2006-01-02 15:04")))
	if includeBody {
		mt.RawSetString("body", lua.LString(m.Body))
	}
	if m.ReplyToID != nil {
		mt.RawSetString("reply_to", lua.LNumber(*m.ReplyToID))
	}
	return mt
}
