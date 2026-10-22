-- hello.lua
--
-- A minimal custom command. This writes only to AnsiChat; it does not send
-- anything to IRC.

ansichat.command("hello", function(ctx)
    local target = ctx.target

    if target == "" or target == "*server*" then
        ansichat.print("Hello from Lua!", ctx.server, "*server*")
        return
    end

    ansichat.print(
        "Hello from Lua in " .. target .. "!",
        ctx.server,
        target
    )
end)
