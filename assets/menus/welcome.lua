-- welcome.lua - Pre-login welcome screen
local menu = {}

function menu.on_enter(node)
    -- Auto-advance to login after displaying.
    -- Set navigation first so even if pause is unavailable/errors, we don't loop.
    node:goto_menu("login")
    if node.pause then
        node:pause()
    end
end

return menu
