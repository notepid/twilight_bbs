-- user_stats.lua - Display current user's statistics
local menu = {}

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    node:pause()
    node:goto_menu("main_menu")
end

return menu
