-- goodbye.lua - Logoff screen
local menu = {}

function menu.on_enter(node)
    node:pause()
    node:disconnect()
end

return menu
