# Menu Scripting

This document describes how to create Lua menu scripts for the BBS.

## Menu File Structure

Menus are defined as file triplets in `assets/menus/`:

- `menu_name.ans` - ANSI color display (preferred)
- `menu_name.asc` - Plain ASCII fallback
- `menu_name.lua` - Lua script with key/input handlers

## Lua Script Structure

```lua
local menu = {}

function menu.on_load(node)
    -- called BEFORE menu art is displayed
    -- use this to clear screen or set initial state
    node:cls()
end

function menu.on_enter(node)
    -- called AFTER menu art is displayed
end

function menu.on_key(node, key)
    -- called on single keypress (hotkey mode)
    if key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

function menu.on_input(node, input)
    -- called when user types string + Enter
end

function menu.on_exit(node)
    -- called when leaving the menu
end

return menu
```

## See Also

- [Lua API Reference](./lua_api.md) - Complete API documentation for all available functions
- [Menu Placeholders](./menu_placeholders.md) - Using `{{...}}` placeholders in menu art files
