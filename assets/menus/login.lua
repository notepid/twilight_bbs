-- login.lua - User login/registration screen
local menu = {}

function menu.on_enter(node)
    node:sendln("")
    local username = node:ask("  Username (NEW for new user): ", 30)
    if username == nil or username == "" then
        node:sendln("")
        node:sendln("  No username entered.")
        node:disconnect()
        return
    end

    if string.upper(username) == "NEW" then
        do_register(node)
        return
    end

    local password = node:password()
    if password == nil or password == "" then
        node:sendln("  No password entered.")
        node:disconnect()
        return
    end

    local user, err = users.login(username, password)
    if user == nil then
        node:sendln("")
        node:sendln("  Login failed: " .. (err or "unknown error"))
        node:sendln("")
        node:pause()
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
    node:pause()
    node:goto_menu("main_menu")
end

function do_register(node)
    node:sendln("")
    node:sendln("  -- New User Registration --")
    node:sendln("")

    local username = node:ask("  Choose a username: ", 30)
    if username == nil or username == "" then
        node:sendln("  Registration cancelled.")
        node:disconnect()
        return
    end

    if users.exists(username) then
        node:sendln("  That username is already taken.")
        node:pause()
        node:goto_menu("login")
        return
    end

    node:sendln("  Password: ")
    local password = node:password()
    if password == nil or password == "" or #password < 4 then
        node:sendln("  Password must be at least 4 characters.")
        node:pause()
        node:goto_menu("login")
        return
    end

    node:sendln("  Confirm password: ")
    local confirm = node:password()
    if confirm ~= password then
        node:sendln("  Passwords do not match.")
        node:pause()
        node:goto_menu("login")
        return
    end

    local real_name = node:ask("  Real name (optional): ", 50)
    local location = node:ask("  Location (optional): ", 50)
    local email = node:ask("  Email (optional): ", 80)

    local user, err = users.register(username, password, real_name, location, email)
    if user == nil then
        node:sendln("  Registration failed: " .. (err or "unknown error"))
        node:pause()
        node:goto_menu("login")
        return
    end

    node:sendln("")
    node:sendln("  Account created! Welcome, " .. user.name .. "!")
    node:sendln("")
    node:pause()
    node:goto_menu("main_menu")
end

return menu
