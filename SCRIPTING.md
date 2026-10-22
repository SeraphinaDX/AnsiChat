# AnsiChat Lua Scripting

AnsiChat embeds Lua through [GopherLua](https://github.com/yuin/gopher-lua). Scripts can add slash commands, react to IRC events, send messages and notices, issue raw IRC commands, and write informational lines into AnsiChat buffers.

Lua support is intended for personal automation and client customization without requiring AnsiChat itself to be rebuilt.

The API is named `ansichat`. Existing scripts using `copperline` keep working:
both globals refer to the same API table. When the new default script directory
is absent, an existing `~/.config/copperline/scripts/` directory is reused.

## Script directory

Scripts are loaded from:

```text
~/.config/ansichat/scripts/
```

All files ending in `.lua` are loaded in alphabetical filename order when AnsiChat starts.

The directory can be changed in `config.toml`:

```toml
[scripting]
enabled = true
dir = "~/.config/ansichat/scripts"
```

Scripting is enabled by default. If the directory does not exist, AnsiChat creates it with mode `0700`. An empty directory has no effect on normal client behavior.

To disable Lua completely:

```toml
[scripting]
enabled = false
```

## Lua commands inside AnsiChat

AnsiChat provides a built-in `/lua` command:

```text
/lua list
/lua reload
/lua eval <code>
```

`/lua list` lists the scripts that loaded successfully.

`/lua reload` discards the current Lua VM, creates a fresh one, and reloads every `.lua` file from the script directory. This means removed globals, commands, and hooks do not linger from an older version of a script.

`/lua eval` evaluates a short piece of Lua in the current VM. It is mainly useful while developing scripts, for example:

```text
/lua eval ansichat.print("hello from Lua")
```

## The `ansichat` API

AnsiChat exposes one global Lua table named `ansichat`.

### `ansichat.command(name, callback)`

Registers a new slash command.

```lua
ansichat.command("hello", function(ctx)
    ansichat.print("Hello from Lua, " .. ctx.nick)
end)
```

The command is then available as:

```text
/hello
```

The callback receives a context table:

```text
ctx.server   current server name
ctx.target   current channel/query/server buffer
ctx.nick     your current nickname on that server
ctx.command  command name
ctx.args     everything typed after the command
```

Example using arguments:

```lua
ansichat.command("slap", function(ctx)
    if ctx.args == "" then
        ansichat.print("usage: /slap nick")
        return
    end

    ansichat.message(
        ctx.target,
        ctx.args .. " gets slapped with a large trout",
        ctx.server
    )
end)
```

AnsiChat's built-in slash commands take precedence over Lua commands. A script cannot replace `/join`, `/msg`, `/quit`, `/lua`, or another built-in command.

### `ansichat.on(event, callback)`

Registers an event hook.

There are two layers of hooks. `irc` receives every raw IRC event:

```lua
ansichat.on("irc", function(ev)
    -- receives every raw IRC event
end)
```

A hook may also target a particular raw IRC command:

```lua
ansichat.on("privmsg", function(ev)
    -- receives PRIVMSG only
end)
```

IRC command names are lower-case in Lua. Examples include:

```text
privmsg
notice
join
part
quit
kick
nick
topic
tagmsg
001
433
```

Numeric replies can therefore be hooked by their three-digit command name.

AnsiChat also emits semantic client events for message types that are more convenient after parsing:

```text
message
action
dcc
startup
```

For example, an IRC CTCP ACTION is available as `action`, so a script does not need to decode the CTCP payload itself:

```lua
ansichat.on("action", function(ev)
    ansichat.print(ev.source .. " performed /me: " .. ev.text)
end)
```

Each event table contains:

```text
ev.server   AnsiChat server name
ev.command  lower-case IRC command
ev.source   nickname/server source
ev.params   Lua array containing IRC parameters
ev.tags     table of IRCv3 message tags
ev.time     Unix timestamp
ev.target   first IRC parameter, when present
ev.text     final IRC parameter when the event has at least two parameters
ev.raw      true for raw IRC events, false for semantic AnsiChat events
```

For example:

```lua
ansichat.on("privmsg", function(ev)
    if ev.text and ev.text:lower():match("hello ansichat") then
        ansichat.print(
            "Lua noticed a greeting from " .. ev.source,
            ev.server,
            ev.target
        )
    end
end)
```

There is also a `startup` event. It runs after all scripts have been loaded or reloaded:

```lua
ansichat.on("startup", function()
    ansichat.print("My scripts are loaded")
end)
```

### `ansichat.print(text [, server [, target]])`

Writes a AnsiChat system line to a buffer without sending anything to IRC.

```lua
ansichat.print("script loaded")
```

If `server` and `target` are omitted, the currently selected buffer is used.

Explicit destination:

```lua
ansichat.print("hello #go-nuts", "libera", "#go-nuts")
```

Lines written with `ansichat.print` go through AnsiChat's normal buffer and logging path.

### `ansichat.message(target, text [, server])`

Sends an IRC `PRIVMSG`.

```lua
ansichat.message("#go-nuts", "hello from Lua")
```

The current server is used when the third argument is omitted.

```lua
ansichat.message("#debian", "hello OFTC", "oftc")
```

### `ansichat.notice(target, text [, server])`

Sends an IRC `NOTICE`.

```lua
ansichat.notice("SomeNick", "hello", "libera")
```

### `ansichat.raw(line [, server])`

Sends a raw IRC protocol line.

```lua
ansichat.raw("WHO #go-nuts")
```

or:

```lua
ansichat.raw("MODE #go-nuts +b", "libera")
```

This is deliberately powerful. Invalid raw IRC can disconnect you, trigger server flood protection, or perform destructive IRC operations. Scripts using `raw` should be treated with the same care as commands typed manually with `/raw`.

### `ansichat.active()`

Returns information about the currently selected buffer:

```lua
local ctx = ansichat.active()

ansichat.print(
    "server=" .. ctx.server ..
    " target=" .. ctx.target ..
    " nick=" .. ctx.nick
)
```

The returned table contains:

```text
server
target
nick
```

### `ansichat.nick([server])`

Returns your current nickname on a server.

```lua
local mynick = ansichat.nick()
```

or:

```lua
local libera_nick = ansichat.nick("libera")
```

## Bundled example scripts

The source tree includes a [`scripts/`](scripts/) directory containing small, commented examples that use the real AnsiChat Lua API:

```text
scripts/hello.lua
scripts/wave.lua
scripts/mentions.lua
scripts/away.lua
scripts/connection.lua
scripts/debug-events.lua
```

They are **not loaded automatically from the repository directory**. Copy only the examples you want into AnsiChat's configured script directory, for example:

```sh
mkdir -p ~/.config/ansichat/scripts
cp scripts/wave.lua ~/.config/ansichat/scripts/
```

Then reload them without restarting AnsiChat:

```text
/lua reload
```

See [`scripts/README.md`](scripts/README.md) for a short description of every example.

## Complete example

Save this as:

```text
~/.config/ansichat/scripts/example.lua
```

```lua
ansichat.on("startup", function()
    ansichat.print("example.lua loaded")
end)

ansichat.command("wave", function(ctx)
    if ctx.target == "*server*" then
        ansichat.print("Select a channel or query first")
        return
    end

    ansichat.message(ctx.target, "o/", ctx.server)
end)

ansichat.on("privmsg", function(ev)
    local mynick = ansichat.nick(ev.server)

    if ev.text and mynick ~= "" and ev.text:lower():match(mynick:lower()) then
        ansichat.print(
            "Lua mention hook: " .. ev.source .. " mentioned you",
            ev.server,
            ev.target
        )
    end
end)
```

Then reload without restarting AnsiChat:

```text
/lua reload
```

and use:

```text
/wave
```

## Concurrency model

AnsiChat's IRC connections, DCC work, notifications, timers, and UI already use Go concurrency. The Lua VM is intentionally different: all Lua callbacks are serialized through one scripting goroutine.

This is important because one GopherLua state should not be called concurrently from unrelated IRC goroutines. AnsiChat therefore queues IRC hook events and executes Lua callbacks one at a time.

IRC processing itself is not allowed to block waiting for Lua. The event queue is bounded; if a script falls extremely far behind, AnsiChat drops scripting hook events rather than stalling the IRC connection.

Custom slash commands are executed synchronously from the UI because their result is user initiated. A badly written command callback that never returns can therefore make the UI appear stuck even though the IRC networking goroutines continue to run.

## Script errors

A script that fails while loading is reported in AnsiChat as a Lua error. AnsiChat never installs a half-loaded replacement VM: if `/lua reload` fails, the previous working set of scripts remains active. On initial startup, a failed load leaves scripting with no newly loaded script set until the files are corrected and reloaded.

Errors raised by an event hook are shown in the active AnsiChat buffer and do not stop later IRC processing.

For custom commands, the error is shown as a normal AnsiChat error line.

After correcting a script, run:

```text
/lua reload
```

## Security

Lua scripts run locally as part of AnsiChat and should be considered trusted code. They can send IRC messages and raw protocol commands using your connected identity.

Do not install random scripts without reading them first.

The scripting API is intended as a convenience boundary, not as a security sandbox.

## Lua compatibility

AnsiChat uses GopherLua, which implements Lua 5.1 semantics with selected additions. Scripts should generally be written as Lua 5.1 code.
