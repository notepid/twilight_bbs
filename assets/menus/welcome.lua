-- welcome.lua - Pre-login welcome screen
local menu = {}

function menu.on_enter(node)
    node:pause()
    node:cls()
    node:goto_menu("login")
end

return menu
