-- connection.lua
--
-- Numeric 001 is the IRC welcome reply. This demonstrates hooking one raw IRC
-- numeric without enabling the very noisy catch-all IRC hook.

ansichat.on("001", function(ev)
    local nick = ansichat.nick(ev.server)
    if nick == "" then
        nick = "unknown nick"
    end

    ansichat.print(
        "Lua noticed that this connection is ready as " .. nick,
        ev.server,
        "*server*"
    )
end)
