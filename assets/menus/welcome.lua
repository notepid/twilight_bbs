-- welcome.lua - Pre-login welcome screen
local menu = {}

function menu.on_enter(node)
    -- Auto-advance to login after displaying
    node:pause()
    node:goto_menu("login")
end

return menu
