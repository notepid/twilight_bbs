-- message_menu.lua - Message bases menu
local menu = {}

function menu.on_load(node)
    node:cls()
end

function menu.on_key(node, key)
    if key == "L" or key == "l" then
        node:goto_menu("message_areas")
    elseif key == "R" or key == "r" then
        node:goto_menu("message_list")
    elseif key == "P" or key == "p" then
        node:goto_menu("message_post")
    elseif key == "S" or key == "s" then
        node:goto_menu("message_scan")
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

return menu
