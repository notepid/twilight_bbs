# Menu Art Placeholders & In-Place Editing

This document explains how to use `{{...}}` placeholders inside menu art files (`.ans` / `.asc`) and how to read/edit those fields from Lua menus.

## Overview

- Placeholders are embedded directly in art files as **literal text** like `{{USER,30}}`.
- When the art is displayed, placeholders are **blanked (replaced with spaces)** so users don’t see the marker text.
- The BBS indexes placeholder positions (row/col) so Lua scripts can move the cursor to the right spot and read input “in place”.

## Placeholder syntax

### Field placeholders

- **Unbounded (uses default length)**:
  - `{{USER}}`
- **With max length (width)**:
  - `{{USER,30}}`
- **With width and height (output regions)**:
  - `{{AREA_LIST,78,18}}`

Rules:
- `ID` is case-sensitive (e.g. `USER` and `User` are different).
- `maxLen` must be a positive integer to be recognized.
- `height` (when present) must be a positive integer to be recognized.
- Backward compatible rule: `{{ID,width}}` implies `height = 1`.
- The placeholder’s **top-left** position (the first `{`) is used as the field coordinate.

### Value placeholders (auto-filled)

Some placeholder IDs are **auto-filled by the server** after the art is displayed. These are printed at the placeholder’s coordinates (using ANSI cursor addressing), and optionally padded/truncated to the placeholder’s `maxLen`.

Examples:

- `{{USERNAME}}`
- `{{USERNAME,12}}` (pad/trim to 12 characters)
- `{{LAST_ON,16}}`

Built-in value IDs:

- `USERNAME` (alias: `NAME`)
- `REAL_NAME`
- `LOCATION`
- `EMAIL`
- `LEVEL` (aliases: `SECURITY_LEVEL`)
- `CALLS` (aliases: `TOTAL_CALLS`)
- `LAST_ON` (formatted like `YYYY-MM-DD HH:MM` when known)
- `CREATED` (formatted like `YYYY-MM-DD`)
- `UPDATED` (formatted like `YYYY-MM-DD`)
- `NODE_ID`
- `NOW` (formatted like `YYYY-MM-DD HH:MM`)

### Special placeholder: `{{CURSOR}}`

- `{{CURSOR}}` moves the terminal cursor to that position **after** the art is displayed and fields are indexed.
- Useful for positioning prompts without hardcoding coordinates.

## How placeholders render

When a menu art file is displayed:
- The content is sent to the terminal as usual (ANSI/VT100 sequences included for `.ans`).
- Any `{{...}}` placeholder sequences are replaced with **spaces of the same length** while rendering.
- After rendering, built-in **value placeholders** (like `{{USERNAME,12}}`) are printed (“overlaid”) at their placeholder positions.

This means:
- The sysop should design the art with enough blank area for typing.
- Users won’t see the placeholder marker text.

## Lua API: field lookup

### `node:field(id)`

Returns the field position for the most recently displayed art.

- **Return**: `row, col, maxLen` or `nil` if not found

Example:

```lua
local row, col, maxLen = node:field("USER")
if row then
  node:goto_xy(row, col)
end
```

## Lua API: in-place input helpers

These helpers are designed for menus that use placeholders.

### `node:input_field(id [, maxLenOverride])`

- Moves to the placeholder position.
- Clears the input area with spaces (width = override or placeholder maxLen or default).
- Reads an unmasked input line.
- **Return**: string or `nil` if the placeholder was not found or on disconnect.

Example:

```lua
local username = node:input_field("USER", 30)
if username == nil then
  -- fallback (no placeholder in art)
  username = node:ask("Username: ", 30)
end
```

### `node:password_field(id [, maxLenOverride])`

Same as `input_field`, but reads a password (no echo / masked).

Example:

```lua
local password = node:password_field("PASS", 30)
if password == nil then
  node:send("Password: ")
  password = node:password()
end
```

### `node:edit_field(id [, maxLenOverride])`

Low-level helper that moves the cursor and reads input, but **does not** clear the placeholder area first.

Use this if you want custom clearing/redraw behavior.

## Lua API: output helpers

### `node:output_field(id, text [, widthOverride [, heightOverride]])`

Print text into a placeholder region (useful for rendering lists and message bodies into an `.asc` template).

- Moves to the placeholder position.
- Clears the entire output rectangle with spaces (width = override or placeholder width; height = override or placeholder height).
- Splits `text` by newlines and prints it into the rectangle.
- **Overflow behavior**:
  - No wrapping.
  - Each line is truncated to `width`.
  - Extra lines beyond `height` are clipped.

Example:

```lua
local rows = {
  "1   General                        120   3",
  "2   Announcements                   42   0",
}
node:output_field("AREA_LIST", table.concat(rows, "\n"))
```

## Lua API: cursor helpers

These only do anything when `node.ansi == true`.

- `node:save_cursor()` / `node:restore_cursor()`
  - Useful when you do in-place editing but want to print messages elsewhere without overwriting art.
- `node:cursor_off()` / `node:cursor_on()`
  - Hide/show the blinking cursor when you’re not expecting input.

Typical pattern:

```lua
-- after art is displayed
node:cursor_off()
node:save_cursor()

-- do in-place input
node:cursor_on()
local v = node:input_field("USER", 30)
node:cursor_off()

-- print messages where the art left the cursor
node:restore_cursor()
node:sendln("")
node:sendln("Thanks!")
```

## Practical notes / limitations

- **ANSI required for positioning**: placeholders and cursor movement only really make sense when `node.ansi` is true.
- **No full terminal emulation**: the placeholder indexer handles common ANSI cursor movement sequences, but not every possible control sequence.
- **“Most recently displayed art”**: field lookup is based on the last display shown (menu display or `node:display(...)`). If you display something else, the field map updates.

