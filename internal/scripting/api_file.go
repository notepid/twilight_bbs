package scripting

import (
	"fmt"

	"github.com/notepid/twilight_bbs/internal/filearea"
	"github.com/notepid/twilight_bbs/internal/user"
	lua "github.com/yuin/gopher-lua"
)

// FileAPI exposes file area functions to Lua.
type FileAPI struct {
	repo        *filearea.Repo
	currentUser func() *user.User
}

// NewFileAPI creates a Lua file area API.
func NewFileAPI(repo *filearea.Repo, currentUser func() *user.User) *FileAPI {
	return &FileAPI{repo: repo, currentUser: currentUser}
}

// Register installs file functions in the Lua state.
func (api *FileAPI) Register(L *lua.LState) {
	mod := L.NewTable()

	mod.RawSetString("areas", L.NewFunction(api.luaAreas))
	mod.RawSetString("get_area", L.NewFunction(api.luaGetArea))
	mod.RawSetString("list", L.NewFunction(api.luaList))
	mod.RawSetString("get_file", L.NewFunction(api.luaGetFile))
	mod.RawSetString("search", L.NewFunction(api.luaSearch))
	mod.RawSetString("add_entry", L.NewFunction(api.luaAddEntry))
	mod.RawSetString("increment_download", L.NewFunction(api.luaIncrementDownload))

	L.SetGlobal("files", mod)
}

func (api *FileAPI) luaAreas(L *lua.LState) int {
	u := api.currentUser()
	level := 0
	if u != nil {
		level = u.SecurityLevel
	}

	areas, err := api.repo.ListAreas(level)
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
		at.RawSetString("files", lua.LNumber(a.FileCount))
		at.RawSetString("download_level", lua.LNumber(a.DownloadLevel))
		at.RawSetString("upload_level", lua.LNumber(a.UploadLevel))
		at.RawSetString("path", lua.LString(a.DiskPath))
		tbl.RawSetInt(i+1, at)
	}
	L.Push(tbl)
	return 1
}

func (api *FileAPI) luaGetArea(L *lua.LState) int {
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
	at.RawSetString("path", lua.LString(a.DiskPath))
	at.RawSetString("upload_level", lua.LNumber(a.UploadLevel))
	at.RawSetString("download_level", lua.LNumber(a.DownloadLevel))
	L.Push(at)
	return 1
}

func (api *FileAPI) luaList(L *lua.LState) int {
	areaID := L.CheckInt(1)
	offset := L.OptInt(2, 0)
	limit := L.OptInt(3, 20)

	entries, err := api.repo.ListFiles(areaID, offset, limit)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, e := range entries {
		tbl.RawSetInt(i+1, api.entryToTable(L, e))
	}
	L.Push(tbl)
	return 1
}

func (api *FileAPI) luaGetFile(L *lua.LState) int {
	fileID := L.CheckInt(1)
	e, err := api.repo.GetFile(fileID)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(api.entryToTable(L, e))
	return 1
}

func (api *FileAPI) luaSearch(L *lua.LState) int {
	pattern := L.CheckString(1)
	u := api.currentUser()
	level := 0
	if u != nil {
		level = u.SecurityLevel
	}

	entries, err := api.repo.FindByName(pattern, level)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	tbl := L.NewTable()
	for i, e := range entries {
		tbl.RawSetInt(i+1, api.entryToTable(L, e))
	}
	L.Push(tbl)
	return 1
}

func (api *FileAPI) luaAddEntry(L *lua.LState) int {
	u := api.currentUser()
	if u == nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("not logged in"))
		return 2
	}

	areaID := L.CheckInt(1)
	filename := L.CheckString(2)
	description := L.OptString(3, "")
	sizeBytes := L.OptInt64(4, 0)

	// Validate filename to prevent path traversal
	validator := &ValidateInput{}
	if err := validator.ValidateFilename(filename); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	if err := validator.ValidateString(description, "description", 255); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	id, err := api.repo.AddEntry(areaID, filename, description, sizeBytes, u.ID)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LNumber(id))
	L.Push(lua.LNil)
	return 2
}

func (api *FileAPI) luaIncrementDownload(L *lua.LState) int {
	fileID := L.CheckInt(1)
	if err := api.repo.IncrementDownload(fileID); err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func (api *FileAPI) entryToTable(L *lua.LState, e *filearea.Entry) *lua.LTable {
	t := L.NewTable()
	t.RawSetString("id", lua.LNumber(e.ID))
	t.RawSetString("area_id", lua.LNumber(e.AreaID))
	t.RawSetString("filename", lua.LString(e.Filename))
	t.RawSetString("description", lua.LString(e.Description))
	t.RawSetString("size", lua.LNumber(e.SizeBytes))
	t.RawSetString("size_str", lua.LString(formatSize(e.SizeBytes)))
	t.RawSetString("uploader", lua.LString(e.UploaderName))
	t.RawSetString("downloads", lua.LNumber(e.DownloadCount))
	t.RawSetString("date", lua.LString(e.UploadedAt.Format("2006-01-02")))
	return t
}

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
