-- user_stats.lua - Display current user's statistics
local menu = {}

function menu.on_enter(node)
    local user = users.get_current()
    if user == nil then
        node:sendln("  Not logged in.")
    else
        node:sendln("")
        node:sendln("  Username:       " .. user.name)
        node:sendln("  Real name:      " .. (user.real_name or ""))
        node:sendln("  Location:       " .. (user.location or ""))
        node:sendln("  Email:          " .. (user.email or ""))
        node:sendln("  Security level: " .. user.level)
        node:sendln("  Total calls:    " .. user.calls)
        if user.last_on then
            node:sendln("  Last on:        " .. user.last_on)
        end
        node:sendln("  Member since:   " .. (user.created or ""))
    end
    node:sendln("")
    node:pause()
    node:goto_menu("main_menu")
end

return menu
