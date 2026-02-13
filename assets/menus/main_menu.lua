-- main_menu.lua - Main BBS menu
local menu = {}

function menu.on_load(node)
    node:cls()
end

function menu.on_key(node, key)
    if key == "M" or key == "m" then
        node:goto_menu("message_menu")
    elseif key == "F" or key == "f" then
        node:goto_menu("file_menu")
    elseif key == "C" or key == "c" then
        node:enter_chat()
    elseif key == "D" or key == "d" then
        node:goto_menu("door_menu")
    elseif key == "W" or key == "w" then
        node:show_online()
        node:goto_menu("main_menu")
    elseif key == "Y" or key == "y" then
        node:goto_menu("user_stats")
    elseif key == "!" then
        node:goto_menu("sysop_menu")
    elseif key == "G" or key == "g" then
        node:goto_menu("goodbye")
    end
end

function menu.on_input(node, input)
    if input == "/who" then
        node:show_online()
        node:goto_menu("main_menu")
    elseif input == "/quit" or input == "/q" then
        node:goto_menu("goodbye")
    end
end

return menu
