-- login.lua - User login screen
local menu = {}

function menu.on_enter(node)
    -- Save where the art left the cursor so we can print messages cleanly
    -- after doing in-place field editing.
    node:save_cursor()

    local username = nil

    -- Prefer placeholders in the art file (e.g. {{USER,30}} / {{PASS,30}}).
    username = node:input_field("USER", 30)
    if username == nil then
        node:sendln("")
        username = node:ask("  Username (NEW for new user): ", 30)
    end

    if username == nil or username == "" then
        node:sendln("")
        node:sendln("  No username entered.")
        node:disconnect()
        return
    end

    if string.upper(username) == "NEW" then
        node:restore_cursor()
        node:goto_menu("registration")
        return
    end

    local password = nil
    password = node:password_field("PASS", 30)
    if password == nil then
        node:send("  Password: ")
        password = node:password()
    end
    if password == nil or password == "" then
        node:restore_cursor()
        node:sendln("  No password entered.")
        node:disconnect()
        return
    end

    node:restore_cursor()

    local user, err = users.login(username, password)
    if user == nil then
        node:sendln("")
        node:sendln("  Login failed: " .. (err or "unknown error"))
        node:sendln("")
        node:send("  Create a new account? [Y/N]: ")
        local choice = node:getkey()
        node:sendln("")
        if string.upper(choice) == 'Y' then
            node:goto_menu("registration")
            return
        end
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
