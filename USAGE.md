# AnsiChat User Guide

[Back to README](README.md)

For a first connection, see the [quick start](README.md#quick-start).
The commands below describe the terminal client; the [GUI guide](GUI.md#using-chat)
lists commands supported by the desktop and Android clients.

## Layout and controls

The left panel contains servers and channel/query buffers. Every visible buffer is numbered. The middle panel contains the current conversation. Wide terminals also get a nick list on the right. The topic sits above the transcript, and the message input and status/control area sit at the bottom.

All action bindings are configurable under `[keybindings]`. Default controls include:

- `Ctrl-N`: next buffer
- `Ctrl-P`: previous buffer
- `Alt-A`: next unread buffer
- `F3` / `F4`: next / previous search match
- `Alt-N`: scroll the right-hand user list down
- `Alt-P`: scroll the right-hand user list up
- `F6`, number, `Enter`: jump directly to a numbered buffer
- `Escape`: cancel buffer jump
- `Tab`: nickname completion; repeated Tab cycles matches
- `Up` / `Down`: older/newer input history for the current buffer
- `PageUp` / `PageDown`: transcript by page
- `Alt-K` / `Alt-J`: transcript by one line
- `Alt-L`: bare/copy mode for terminal text selection
- `End`: return to/follow newest messages
- `Alt-R`: force reconnect to a AnsiChat relay while in relay-client mode
- `Ctrl-U`: clear input
- `Ctrl-C`: quit

For example:

```toml
[keybindings]
user_list_down = "Ctrl+Y"
user_list_up = "Ctrl+U"
history_previous = "Up"
history_next = "Down"
relay_reconnect = "Alt+R"
```

If you reassign a binding such as `Ctrl+U`, move any conflicting action to another key. AnsiChat validates configurable action keys so invalid or conflicting bindings fail clearly instead of silently doing something unexpected.

Input history is session-only and separate for each buffer. It stores both ordinary messages and slash commands. The default is 10 entries per buffer and is configurable with:

```toml
[general]
input_history_limit = 10
```

Set it to `0` to disable input history.

Press `Alt+B` to show/hide the channel list, or `Alt+U` to show/hide the user list. Both toggles keep chat live and your draft intact.

Press `Alt-L` for bare/copy mode. AnsiChat temporarily hides the normal UI chrome and disables terminal mouse reporting so the terminal can perform normal text selection. IRC, logging, Gotify, DCC, scripting, and incoming message state continue in the background.

## Commands

```text
/help
/buffer number
/quick [-tls|-plain] host[:port] [port] [nickname]
/server [name]
/connect [server]
/disconnect [server] [reason]
/join #channel [key]
/part [#channel] [reason]
/query nick
/msg nick message
/me action
/notice target message
/ctcp nick command [text]
/nick newnick
/oper name password
/away [message]
/back
/mode [target] [modes [arguments]]
/op [#channel] nick
/deop [#channel] nick
/voice [#channel] nick
/devoice [#channel] nick
/kick [#channel] nick [reason]
/ban [#channel] [mask]
/unban [#channel] mask
/invite nick [#channel]
/list [channels [server]]
/names [#channel]
/who [mask [flags]]
/whowas nick [count [server]]
/motd [server]
/time [server]
/stats [query [server]]
/links [mask | server mask]
/map
/flex
/topic new topic
/whois nick
/ignore [list]
/ignore add mask [--channel|--global]
/ignore remove number|mask
/ignore clear
/raw IRC COMMAND HERE
/history [count]
/search [text]
/searchnext
/searchprev
/unread
/markread
/caps
/dcc list
/dcc accept nick
/dcc send nick /path/to/file
/paste status
/paste cancel
/paste restore
/paste discard
/gotify status
/gotify test
/lua list
/lua reload
/lua eval <code>
/close
/quit [reason]
```

Channel commands default to the active channel; outside a channel, supply one
explicitly. `/mode` defaults to the active channel or, in other buffers, your own
nickname. `/mode +m` changes the current channel; `/mode yournick +i` changes your
user modes. A leading `+` or `-` is interpreted as a mode string; use `/raw MODE`
to explicitly address legacy `+channel` names. `/ban` without a mask lists bans;
ban masks are sent as entered (there is no automatic nick-to-hostmask lookup).
`/invite #channel nick` is also accepted. `/who` defaults to the active channel,
or `*` elsewhere. `/map` requests the current server's network topology and displays
its replies in that server's status buffer. `/names`, `/who`, and channel-mode query replies appear in the
requesting buffer; other server replies use the existing server/channel output.

For `/oper`, everything after the name is the password, including internal spaces
and a leading colon; do not add quotes. The command is excluded from input history.
`/away` with no message and `/back` both clear away status.

CTCP examples:

```text
/ctcp Alice VERSION
/ctcp Alice TIME
/ctcp Alice PING 123456789
```

## Temporary connections with /quick

Connect to an unconfigured IRC server with `/quick irc.example.org`. TLS with
certificate verification is the default (port 6697). Use
`/quick irc.example.org 6697 MyNick` or `/quick irc.example.org:6697 MyNick`
for a custom port/nickname, or `/quick -plain irc.example.org 6667` for plaintext.
Without a nickname override, the connection uses the IRC-owning process's
`[general]` identity. Network passwords, SASL, client certificates, and autojoin
channels are not inherited.

The command selects the new server's status buffer; use `/join #channel` next.
Sessions remain available for `/connect` and `/disconnect` until the process
exits, without writing to TOML. Repeating the same endpoint and nickname reuses
the session. In relay mode the connection is made on the relay and shared by
attached clients; update both client and relay to 0.2.37 to use `/quick`.
The wire protocol remains version 2; older relays reject the new command.
