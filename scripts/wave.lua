-- wave.lua
--
-- Adds /wave. In a channel or private query it sends a small greeting.

ansichat.command("wave", function(ctx)
    if ctx.server == "" then
        ansichat.print("/wave: no active server")
        return
    end

    if ctx.target == "" or ctx.target == "*server*" then
        ansichat.print("/wave: select a channel or private query first")
        return
    end

    ansichat.message(ctx.target, "o/", ctx.server)
end)
