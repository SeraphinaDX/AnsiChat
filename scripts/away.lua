-- away.lua
--
-- Adds simple away aliases using raw IRC.
--
--   /brb
--   /brb making tea
--   /back

ansichat.command("brb", function(ctx)
    if ctx.server == "" then
        ansichat.print("/brb: no active server")
        return
    end

    local reason = ctx.args
    if reason == "" then
        reason = "Away"
    end

    ansichat.raw("AWAY :" .. reason, ctx.server)
    ansichat.print("Away: " .. reason, ctx.server, "*server*")
end)

ansichat.command("back", function(ctx)
    if ctx.server == "" then
        ansichat.print("/back: no active server")
        return
    end

    ansichat.raw("AWAY", ctx.server)
    ansichat.print("Away status cleared", ctx.server, "*server*")
end)
