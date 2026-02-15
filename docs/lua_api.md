# Lua API Reference

This document describes all available functions in the `node` object for writing BBS menu scripts in Lua.

## Table of Contents

- [Output Functions](#output-functions)
- [Input Functions](#input-functions)
- [Field Functions](#field-functions)
- [Navigation Functions](#navigation-functions)
- [State Functions](#state-functions)
- [Terminal Functions](#terminal-functions)
- [Inter-node Functions](#inter-node-functions)
- [Pre-authentication Functions](#pre-authentication-functions)
- [Properties](#properties)
- [User API](#user-api)
- [Message API](#message-api)
- [File Area API](#file-area-api)
- [Chat API](#chat-api)
- [Transfer API](#transfer-api)
- [Door API](#door-api)

---

## Output Functions

### `node:send(text)`

Sends text to the terminal without a newline.

- **Parameters:**
  - `text` (string): Text to send
- **Returns:** none

### `node:sendln(text)`

Sends text to the terminal with a newline.

- **Parameters:**
  - `text` (string): Text to send (optional)
- **Returns:** none

### `node:log(text)`

Logs text to the server console.

- **Parameters:**
  - `text` (string): Text to log
- **Returns:** none

### `node:cls()`

Clears the terminal screen.

- **Returns:** none

### `node:display(menuName)`

Displays a menu art file without running its script.

- **Parameters:**
  - `menuName` (string): Name of the menu to display (without extension)
- **Returns:** none

---

## Input Functions

### `node:getkey()`

Waits for and returns a single keypress.

- **Returns:** string (the key character)

### `node:getline(maxLen)`

Reads a line of input with echo.

- **Parameters:**
  - `maxLen` (number): Maximum length
- **Returns:** string (the entered text)

### `node:ask(prompt, maxLen)`

Displays a prompt and reads a line of input.

- **Parameters:**
  - `prompt` (string): Prompt text to display
  - `maxLen` (number): Maximum length
- **Returns:** string or nil (if disconnected)

### `node:password()`

Reads a password without echo.

- **Returns:** string or nil (if disconnected)

### `node:yesno(prompt)`

Displays a prompt and waits for Y or N.

- **Parameters:**
  - `prompt` (string): Prompt text to display
- **Returns:** boolean (true for Y, false for N)

### `node:hotkey(keys)`

Waits for one of several specific keys.

- **Parameters:**
  - `keys` (string): String of allowed keys (e.g., "QRM")
- **Returns:** string (the key that was pressed)

### `node:pause([seconds])`

Displays "Press any key to continue..." and waits for a keypress.

- **Parameters:**
  - `seconds` (number, optional): If provided, automatically continues after this many seconds. Displays a countdown.
- **Returns:** none

---

## Field Functions

These functions work with `{{...}}` placeholders in menu art files. See [Menu Art Placeholders](./menu_placeholders.md) for details.

### `node:field(id)`

Returns the position of a placeholder field.

- **Parameters:**
  - `id` (string): Field identifier (e.g., "USER")
- **Returns:** `row, col, maxLen` or `nil` if not found

### `node:input_field(id [, maxLen])`

Moves to a placeholder position and reads input with echo.

- **Parameters:**
  - `id` (string): Field identifier
  - `maxLen` (number, optional): Override max length
- **Returns:** string or nil

### `node:password_field(id [, maxLen])`

Moves to a placeholder position and reads a password (masked).

- **Parameters:**
  - `id` (string): Field identifier
  - `maxLen` (number, optional): Override max length
- **Returns:** string or nil

### `node:edit_field(id [, maxLen])`

Low-level field editing - reads input without clearing the field first.

- **Parameters:**
  - `id` (string): Field identifier
  - `maxLen` (number, optional): Override max length
- **Returns:** string or nil

### `node:output_field(id, text [, width, height])`

Prints text into a placeholder region.

- **Parameters:**
  - `id` (string): Field identifier
  - `text` (string): Text to output
  - `width` (number, optional): Override width
  - `height` (number, optional): Override height
- **Returns:** none

---

## Navigation Functions

### `node:goto_menu(name)`

Navigates to a different menu.

- **Parameters:**
  - `name` (string): Menu name (without extension)
- **Returns:** none

### `node:gosub_menu(name)`

Calls a submenu, returning to the current menu afterward.

- **Parameters:**
  - `name` (string): Menu name
- **Returns:** none

### `node:return_menu()`

Returns from a gosub menu to the caller.

- **Returns:** none

### `node:disconnect()`

Closes the connection and ends the session.

- **Returns:** none

---

## State Functions

### `node:set_state(key, value)`

Sets a session-wide state value (persists across menus).

- **Parameters:**
  - `key` (string): State key
  - `value` (any): State value
- **Returns:** none

### `node:get_state(key)`

Gets a session-wide state value.

- **Parameters:**
  - `key` (string): State key
- **Returns:** any (or nil)

### `node:set_session(key, value)`

Alias for `node:set_state()`.

### `node:get_session(key)`

Alias for `node:get_state()`.

---

## Terminal Functions

### `node:goto_xy(row, col)`

Moves the cursor to a specific position.

- **Parameters:**
  - `row` (number): Row (1-indexed)
  - `col` (number): Column (1-indexed)
- **Returns:** none

### `node:color(colorCode [, bgColor])`

Changes the text color.

- **Parameters:**
  - `colorCode` (number): ANSI foreground color (1=red, 2=green, 3=yellow, 4=blue, 5=magenta, 6=cyan, 7=white, 0=default)
  - `bgColor` (number, optional): ANSI background color (same codes)
- **Returns:** none

### `node:save_cursor()`

Saves the current cursor position.

- **Returns:** none

### `node:restore_cursor()`

Restores a previously saved cursor position.

- **Returns:** none

### `node:cursor_off()`

Hides the cursor (only works when `node.ansi` is true).

- **Returns:** none

### `node:cursor_on()`

Shows the cursor (only works when `node.ansi` is true).

- **Returns:** none

---

## Pre-authentication Functions

These functions provide access to credentials passed during SSH authentication.

### `node:preauth_username()`

Returns the username provided during SSH authentication.

- **Returns:** string (empty if not SSH or not provided)

### `node:preauth_password()`

Returns the password provided during SSH authentication.

- **Returns:** string (empty if not SSH or not provided)

---

## Properties

### `node.width` (read-only)

The terminal width in characters.

- **Type:** number

### `node.height` (read-only)

The terminal height in characters.

- **Type:** number

### `node.ansi` (read-only)

Whether the terminal supports ANSI codes.

- **Type:** boolean

---

## Inter-node Functions

### `node:show_online()`

Shows the list of currently online users (nodes).

- **Returns:** none

### `node:enter_chat()`

Enters the multi-node chat system.

- **Returns:** none

### `node:launch_door(name)`

Launches a door by name.

- **Parameters:**
  - `name` (string): Door name as configured in door_menu.lua
- **Returns:** none

### `node:more([seconds])`

Alias for `node:pause()`. Displays "Press any key to continue..." and waits for a keypress.

- **Parameters:**
  - `seconds` (number, optional): If provided, automatically continues after this many seconds.
- **Returns:** none

---

## User API

The `users` object provides user management functions.

### `users.login(username, password)`

Authenticates a user.

- **Parameters:**
  - `username` (string)
  - `password` (string)
- **Returns:** `user, err` where user is a table with fields: `id`, `username`, `name`, `level`, `calls`, `last_on`

### `users.register(username, password [, realName, location, email])`

Creates a new user account.

- **Parameters:**
  - `username` (string)
  - `password` (string)
  - `realName` (string, optional)
  - `location` (string, optional)
  - `email` (string, optional)
- **Returns:** `user, err`

### `users.exists(username)`

Checks if a username exists.

- **Parameters:**
  - `username` (string)
- **Returns:** boolean

---

## Message API

The `msg` object provides access to the message base system.

### `msg.areas()`

Returns a list of message areas accessible to the current user.

- **Returns:** table of areas, each with: `id`, `name`, `description`, `total`, `new`, `read_level`, `write_level`

### `msg.get_area(areaID)`

Returns details about a specific message area.

- **Parameters:**
  - `areaID` (number)
- **Returns:** table with `id`, `name`, `description`, `total` or `nil` on error

### `msg.list(areaID [, offset, limit])`

Lists messages in an area.

- **Parameters:**
  - `areaID` (number)
  - `offset` (number, optional): Skip this many messages (default 0)
  - `limit` (number, optional): Max messages to return (default 20)
- **Returns:** table of messages, each with: `id`, `area_id`, `from`, `from_id`, `to`, `subject`, `date`

### `msg.read(msgID)`

Reads a specific message and marks it as read.

- **Parameters:**
  - `msgID` (number)
- **Returns:** table with: `id`, `area_id`, `from`, `from_id`, `to`, `subject`, `body`, `date`, `reply_to`

### `msg.post(areaID, subject, body [, to, replyTo])`

Posts a new message to an area.

- **Parameters:**
  - `areaID` (number)
  - `subject` (string): Message subject
  - `body` (string): Message body
  - `to` (string, optional): Recipient name (for private messages)
  - `replyTo` (number, optional): Message ID being replied to
- **Returns:** `msgID, err` - new message ID or nil + error string

### `msg.scan_new(areaID)`

Scans for new (unread) messages in an area.

- **Parameters:**
  - `areaID` (number)
- **Returns:** table of new messages

### `msg.mark_read(areaID, msgID)`

Marks a specific message as read.

- **Parameters:**
  - `areaID` (number)
  - `msgID` (number)
- **Returns:** none

### `msg.count(areaID)`

Returns the total number of messages in an area.

- **Parameters:**
  - `areaID` (number)
- **Returns:** number

---

## File Area API

The `files` object provides access to the file area system.

### `files.areas()`

Returns a list of file areas accessible to the current user.

- **Returns:** table of areas, each with: `id`, `name`, `description`, `files`, `download_level`, `upload_level`, `path`

### `files.get_area(areaID)`

Returns details about a specific file area.

- **Parameters:**
  - `areaID` (number)
- **Returns:** table with `id`, `name`, `description`, `path`, `upload_level`, `download_level`

### `files.list(areaID [, offset, limit])`

Lists files in an area.

- **Parameters:**
  - `areaID` (number)
  - `offset` (number, optional): Skip this many files (default 0)
  - `limit` (number, optional): Max files to return (default 20)
- **Returns:** table of files, each with: `id`, `area_id`, `filename`, `description`, `size`, `size_str`, `uploader`, `downloads`, `date`

### `files.get_file(fileID)`

Returns details about a specific file.

- **Parameters:**
  - `fileID` (number)
- **Returns:** table with file details (same as `files.list`) or `nil`

### `files.search(pattern)`

Searches for files by name pattern.

- **Parameters:**
  - `pattern` (string): Search pattern (supports wildcards)
- **Returns:** table of matching files

### `files.add_entry(areaID, filename, description [, sizeBytes])`

Adds a new file entry to an area (for uploads).

- **Parameters:**
  - `areaID` (number)
  - `filename` (string)
  - `description` (string, optional)
  - `sizeBytes` (number, optional)
- **Returns:** `entryID, err` - new entry ID or nil + error string

### `files.increment_download(fileID)`

Increments the download count for a file.

- **Parameters:**
  - `fileID` (number)
- **Returns:** `err` or `nil` on success

---

## Chat API

The `chat` object provides multi-node chat and messaging functions.

### `chat.send(nodeID, text)`

Sends a private message to another node.

- **Parameters:**
  - `nodeID` (number): Target node number
  - `text` (string): Message text
- **Returns:** `err` or `nil` on success

### `chat.broadcast(text)`

Sends a message to all online users.

- **Parameters:**
  - `text` (string): Message text
- **Returns:** none

### `chat.online()`

Returns a list of all online users.

- **Returns:** table of users, each with: `node_id`, `name`, `room`

### `chat.enter_room(roomName)`

Joins a chat room.

- **Parameters:**
  - `roomName` (string): Room name
- **Returns:** none

### `chat.leave_room()`

Leaves the current chat room.

- **Returns:** none

### `chat.room_members(roomName)`

Returns list of members in a room.

- **Parameters:**
  - `roomName` (string): Room name
- **Returns:** table of member names

### `chat.send_room(roomName, text)`

Sends a message to everyone in a room.

- **Parameters:**
  - `roomName` (string): Room name
  - `text` (string): Message text
- **Returns:** none

---

## Transfer API

The `transfer` object provides ZMODEM file transfer functionality via SEXYZ.

### `transfer.available()`

Checks if file transfers are available (SEXYZ binary found).

- **Returns:** boolean

### `transfer.send(filePaths...)`

Sends one or more files to the client via ZMODEM.

- **Parameters:**
  - `filePaths` (string or table): Single file path, or table of paths, or multiple arguments
- **Returns:** `success, err` - boolean + error string or nil

### `transfer.receive(uploadDir)`

Receives files from the client via ZMODEM.

- **Parameters:**
  - `uploadDir` (string): Directory to save uploaded files
- **Returns:** `filesTable, err` - table of received files with `name` and `size`, or nil + error string

Example:
```lua
local files, err = transfer.receive("/uploads")
if files then
    for i, f in ipairs(files) do
        node:sendln(string.format("Received: %s (%d bytes)", f.name, f.size))
    end
end
```

---

## Door API

The `door` object provides access to DOS door games via DOSEMU2.

### `door.available()`

Checks if doors are available (DOSEMU2 installed).

- **Returns:** boolean

### `door.launch(configTable)`

Launches a door game.

- **Parameters:**
  - `configTable` (table): Door configuration with fields:
    - `name` (string, required): Door name
    - `command` (string, required): DOS command to run
    - `description` (string, optional): Door description
    - `drop_file_type` (string, optional): "DOOR.SYS" or "DORINFO1.DEF" (default: "DOOR.SYS")
    - `security_level` (number, optional): Minimum security level (default: 10)
    - `multiuser` (boolean, optional): Allow concurrent users (default: true). If false, launch is denied while already in use.
- **Returns:** `err` or `nil` on success

Example:
```lua
door.launch({
    name = "Door Game",
    command = "C:\\DOORS\\MYGAME\\GAME.EXE",
    description = "Play the classic game!",
    drop_file_type = "DOOR.SYS",
    security_level = 10,
  multiuser = true,
})
```
