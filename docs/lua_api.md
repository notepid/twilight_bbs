# Lua API Reference

This document describes all available functions in the `node` object for writing BBS menu scripts in Lua.

## Table of Contents

- [Output Functions](#output-functions)
- [Input Functions](#input-functions)
- [Field Functions](#field-functions)
- [Navigation Functions](#navigation-functions)
- [State Functions](#state-functions)
- [Terminal Functions](#terminal-functions)
- [Pre-authentication Functions](#pre-authentication-functions)
- [Properties](#properties)

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

### `node:color(colorCode)`

Changes the text color.

- **Parameters:**
  - `colorCode` (string): ANSI color code (e.g., "1" for red, "32" for green)
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
