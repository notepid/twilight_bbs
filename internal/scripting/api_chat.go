package scripting

import (
	"fmt"

	"github.com/mikael/twilight_bbs/internal/chat"
	"github.com/mikael/twilight_bbs/internal/terminal"
	lua "github.com/yuin/gopher-lua"
)

// ChatAPI exposes inter-node chat functions to Lua.
type ChatAPI struct {
	broker   *chat.Broker
	term     *terminal.Terminal
	nodeID   int
	userName func() string
}

// NewChatAPI creates a Lua chat API.
func NewChatAPI(broker *chat.Broker, term *terminal.Terminal, nodeID int, userName func() string) *ChatAPI {
	return &ChatAPI{
		broker:   broker,
		term:     term,
		nodeID:   nodeID,
		userName: userName,
	}
}

// Register installs chat functions in the Lua state.
func (api *ChatAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("send", L.NewFunction(api.luaSend))
	mod.RawSetString("broadcast", L.NewFunction(api.luaBroadcast))
	mod.RawSetString("online", L.NewFunction(api.luaOnline))
	mod.RawSetString("enter_room", L.NewFunction(api.luaEnterRoom))
	mod.RawSetString("leave_room", L.NewFunction(api.luaLeaveRoom))
	mod.RawSetString("room_members", L.NewFunction(api.luaRoomMembers))
	mod.RawSetString("send_room", L.NewFunction(api.luaSendRoom))

	L.SetGlobal("chat", mod)
}

func (api *ChatAPI) luaSend(L *lua.LState) int {
	toNodeID := L.CheckInt(1)
	text := L.CheckString(2)

	err := api.broker.SendTo(api.nodeID, api.userName(), toNodeID, text)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func (api *ChatAPI) luaBroadcast(L *lua.LState) int {
	text := L.CheckString(1)
	api.broker.Broadcast(api.nodeID, api.userName(), text)
	return 0
}

func (api *ChatAPI) luaOnline(L *lua.LState) int {
	users := api.broker.ListOnline()
	tbl := L.NewTable()
	for i, u := range users {
		ut := L.NewTable()
		ut.RawSetString("node_id", lua.LNumber(u.NodeID))
		ut.RawSetString("name", lua.LString(u.UserName))
		ut.RawSetString("room", lua.LString(u.Room))
		tbl.RawSetInt(i+1, ut)
	}
	L.Push(tbl)
	return 1
}

func (api *ChatAPI) luaEnterRoom(L *lua.LState) int {
	room := L.CheckString(1)
	api.broker.JoinRoom(api.nodeID, room)

	// Announce entry
	api.broker.SendToRoom(api.nodeID, api.userName(), room,
		fmt.Sprintf("*** %s has joined the room ***", api.userName()))

	return 0
}

func (api *ChatAPI) luaLeaveRoom(L *lua.LState) int {
	// Announce departure before leaving
	sub := api.broker.ListOnline()
	for _, u := range sub {
		if u.NodeID == api.nodeID && u.Room != "" {
			api.broker.SendToRoom(api.nodeID, api.userName(), u.Room,
				fmt.Sprintf("*** %s has left the room ***", api.userName()))
			break
		}
	}

	api.broker.LeaveRoom(api.nodeID)
	return 0
}

func (api *ChatAPI) luaRoomMembers(L *lua.LState) int {
	room := L.CheckString(1)
	members := api.broker.RoomMembers(room)

	tbl := L.NewTable()
	for i, name := range members {
		tbl.RawSetInt(i+1, lua.LString(name))
	}
	L.Push(tbl)
	return 1
}

func (api *ChatAPI) luaSendRoom(L *lua.LState) int {
	room := L.CheckString(1)
	text := L.CheckString(2)
	api.broker.SendToRoom(api.nodeID, api.userName(), room, text)
	return 0
}
