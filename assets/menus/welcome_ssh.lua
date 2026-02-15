-- welcome_ssh.lua - SSH welcome screen with pre-authentication
local menu = {}

function menu.on_enter(node)
    node:pause(2)
    node:cls()

    local username = node:preauth_username()
    local password = node:preauth_password()

    if username == "" or password == "" then
        node:goto_menu("welcome")
        return
    end

    node:sendln("")
    node:sendln("  Attempting auto-login as: " .. username)

    local user, err = users.login(username, password)
    if user == nil then
        node:sendln("")
        node:sendln("  Invalid SSH credentials.")
        node:sendln("  You may login interactively or create a new account.")
        node:sendln("")
        node:pause()
        node:cls()
        node:goto_menu("login")
        return
    end

    node:sendln("")
    node:sendln("  Welcome back, " .. user.name .. "!")
    node:sendln("  Security level: " .. user.level)
    node:sendln("  Total calls: " .. user.calls)
    if user.last_on then
        node:sendln("  Last on: " .. user.last_on)
    end
    node:sendln("")
    node:pause(2)
    node:cls()
    node:goto_menu("main_menu")
end

return menu
