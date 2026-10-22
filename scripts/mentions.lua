-- mentions.lua
--
-- Counts messages/actions that contain your current nickname during this
-- AnsiChat session. The counter resets whenever Lua is reloaded.
--
-- Commands:
--   /mentioncount
--   /mentioncount reset

local mention_count = 0

local function inspect_message(ev)
    local mynick = ansichat.nick(ev.server)
    if mynick == "" then
        return
    end

    -- Ignore our own echoed messages/actions.
    if ev.source and ev.source:lower() == mynick:lower() then
        return
    end

    local text = ev.text or ""
    if text:lower():find(mynick:lower(), 1, true) then
        mention_count = mention_count + 1
    end
end

ansichat.on("message", inspect_message)
ansichat.on("action", inspect_message)

ansichat.command("mentioncount", function(ctx)
    if ctx.args:lower() == "reset" then
        mention_count = 0
        ansichat.print("Lua mention counter reset")
        return
    end

    ansichat.print(
        "Lua mention count this session: " .. tostring(mention_count)
    )
end)
