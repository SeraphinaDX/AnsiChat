# AnsiChat Features

[Back to README](README.md)

## Why AnsiChat is different

AnsiChat deliberately combines the things people expect from a traditional IRC client with features that are usually split across a client, a bouncer, shell tools, and notification services.

- **A real native terminal IRC client** — built in Go, compiled to a native executable, and designed around concurrent network work without making the UI wait on it.
- **A built-in SSH relay/bouncer** — run AnsiChat on a server, keep IRC connected there, and attach AnsiChat clients from other machines.
- **No external SSH daemon required** — relay mode has its own embedded SSH server and private AnsiChat channel protocol.
- **No certificate-authority hassle** — relay clients authenticate with SSH public keys and pin the relay's SSH host-key fingerprint.
- **Safer relay message delivery** — chat text is not cleared from the input field until the relay acknowledges the request. If the transport dies or a send cannot be confirmed, the text stays in the input instead of silently disappearing.
- **Relay health monitoring** — the client heartbeats the SSH transport and exposes an always-visible reconnect control with a configurable shortcut.
- **Multiple attached clients** — more than one AnsiChat client can attach to the same relay while the relay maintains the IRC connections.
- **Modern IRCv3 without abandoning classic IRC** — server-time, message tags, echo-message, typing indicators, CHATHISTORY, account/away state, SASL, CTCP, DCC, `/raw`, and the familiar slash-command workflow all live together.
- **A TUI meant to be lived in** — channel topics, nick lists, mouse support, buffer numbers, direct jumps, nickname completion, per-buffer input history, copy mode, startup progress, and configurable keybindings.
- **Lua scripting built in** — add commands, react to IRC events, automate repetitive work, and extend AnsiChat without recompiling it.
- **Notifications without freezing IRC** — Gotify requests run asynchronously and can alert on mentions and private messages.
- **Everything is themeable** — true-color palettes cover buffers, nicks, mentions, message types, borders, input, cursor, status areas, and more.

AnsiChat is designed so the IRC connection, UI, logging, scripting, notifications, relay transport, and DCC code remain separate pieces rather than collapsing into one giant `main.go`.

## Feature highlights

### Native AnsiChat SSH relay

AnsiChat can be its own IRC bouncer.

Run one AnsiChat instance in relay-server mode on an always-on machine. It maintains the real IRC connections, retains bounded in-memory conversation history, tracks channel state, and accepts AnsiChat clients over its **embedded SSH server**.

A relay client runs the normal AnsiChat TUI locally while the remote relay stays connected to IRC. Closing the local client does not make the relay leave your networks and channels.

Relay mode includes:

- public-key-only authentication;
- automatically generated Ed25519 host and client keys;
- SHA-256 host-key fingerprint pinning;
- a AnsiChat-specific `authorized_keys` file;
- multiple attached AnsiChat clients;
- retained message replay when a client attaches again;
- synchronized server, channel, topic, nick-list, capability, typing, and connection state;
- suppression of huge WHO/NAMES housekeeping floods such as `352`, `353`, and `354` from the relay transport;
- an SSH heartbeat so stale connections stop pretending to be healthy;
- a dedicated clickable **RECONNECT** control in relay-client mode;
- **Alt+R** as the default configurable force-reconnect shortcut;
- safe force reconnect that replaces only the local SSH attachment and leaves IRC connected on the relay;
- relay send acknowledgement before AnsiChat clears normal chat text from the input field.

The embedded SSH endpoint is intentionally narrow. It does **not** provide a shell, PTY, SFTP, remote command execution, or SSH port forwarding. It accepts AnsiChat's private relay channel only.

See [`RELAY.md`](RELAY.md) for complete setup instructions.

### Modern IRCv3

AnsiChat requests a broad modern capability set while still behaving like a recognizable IRC client. Depending on what the IRC network supports, AnsiChat can use:

- CAP negotiation;
- message tags;
- server-time;
- echo-message;
- account-notify and account-tag;
- away-notify;
- chghost;
- extended-join;
- multi-prefix;
- userhost-in-names;
- labeled-response;
- standard replies;
- CHATHISTORY;
- event playback;
- read markers;
- IRCv3 `+typing` client tags;
- SASL PLAIN and EXTERNAL.

The negotiated capability set is visible with `/caps`, and `/raw` remains available for network-specific features and experiments.

### A responsive, practical TUI

AnsiChat's interface is designed for long-running everyday use rather than a minimal demo.

- multiple IRC networks at once;
- channels and private-query buffers;
- numbered buffers;
- direct buffer jumping with `F6`;
- next/previous buffer shortcuts;
- channel topics with wrapping;
- scrollable nick list;
- mouse wheel support for transcripts and nick lists;
- double-click a nick to open a private query;
- channel-aware nickname completion with Tab cycling;
- `Up` / `Down` per-buffer message and command history;
- configurable input-history limit, defaulting to 10 entries per buffer;
- page and single-line transcript scrolling;
- WeeChat-style bare/copy mode for easy terminal text selection;
- visible startup progress while servers connect and channels join;
- drafts kept in the input when AnsiChat is not ready to send them;
- configurable keybindings with collision and validation checks;
- compatibility handling for modern terminal Ctrl/Alt key event forms.

### Lua scripting

AnsiChat embeds Lua for client-side extension and automation. Scripts can:

- register custom slash commands;
- receive IRC event hooks;
- send messages and notices;
- write local buffer text;
- send raw IRC commands;
- inspect the active server, target, and nickname;
- reload without restarting AnsiChat.

Use `/lua reload` to reload scripts. AnsiChat performs a physical terminal resync after script reloads to keep the TUI clean.

See [`SCRIPTING.md`](SCRIPTING.md) for the complete scripting API and [`scripts/`](scripts/) for examples.

### DCC, Gotify, logging, and themes

AnsiChat also includes features that are often missing from smaller terminal IRC clients:

- active DCC SEND;
- incoming DCC CHAT;
- asynchronous Gotify notifications for mentions and private messages;
- per-network/per-buffer persistent logs;
- configurable date/time stamps;
- muted pre-session conversation backlog without confusing current-session messages with old log text;
- configurable mention highlighting;
- deterministic per-nick colors;
- full true-color theme customization.

## Configurable channel departure notices

`[general].show_part_messages` and `[general].show_quit_messages` independently
control PART and QUIT transcript notices. Both default to `true`. QUIT notices
appear in the server buffer and in each shared channel before membership is
removed; neither departure event increments unread activity. In relay mode,
set these options on the IRC-owning relay server and restart it. The relay
forwards generated notices to attached clients without a protocol change.

## IRCv3 capability set

Capabilities currently requested by default include:

```text
account-notify
account-tag
away-notify
batch
cap-notify
chghost
echo-message
extended-join
invite-notify
labeled-response
message-tags
multi-prefix
server-time
setname
standard-replies
userhost-in-names
draft/chathistory
draft/event-playback
draft/extended-monitor
draft/message-redaction
draft/multiline
draft/no-implicit-names
draft/read-marker
```

Not every requested draft capability has a dedicated UI action yet. `/raw` remains available for experimenting with network-specific IRCv3 commands and extensions.

## DCC notes

Traditional DCC establishes a direct TCP connection between IRC users. It is not protected by the TLS connection to the IRC server and can reveal IP addresses.

For outgoing DCC SEND, configure `dcc.advertise_ip` to an address the other user can reach. NAT/router port forwarding may be necessary. Incoming filenames are reduced to their basename, downloads use exclusive file creation, and existing files are not overwritten.

AnsiChat currently implements normal active DCC SEND and incoming DCC CHAT. Passive/reverse DCC SEND (`port 0`) and DCC RESUME are possible future additions.

## Mention highlighting

AnsiChat highlights incoming messages and `/me` actions that mention your current nickname. The mention color is configurable with `theme.mention`. Your own outgoing messages are never highlighted as mentions.

Unread mentions also highlight channels you are not viewing until you open the
channel or mark the mention read.

## Screenshots

![Terminal themes](screenshots/screenshot2.avif)

![Android client](screenshots/screenshot6.avif)

![Gotify notifications](screenshots/screenshot3.avif)
