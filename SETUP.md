# AnsiChat Setup Guide

[Back to README](README.md)

## Requirements

AnsiChat requires **Go 1.26 or newer**. Relay mode uses the September 2026 SSH security fixes in `golang.org/x/crypto v0.56.0`.

## Build

```sh
go build -o ansichat ./cmd/ansichat
```

Check the version with:

```sh
./ansichat -version
```

## Configuration

### Upgrading from Copperline

AnsiChat 0.2.32 continues the existing project under its new name. Build the
renamed entry points with `make tui` and `make gui`, or use the commands below.
For an installed desktop launcher, run `make install-gui` to update its name and
executable. Update any relay service or startup command to run `ansichat`.

AnsiChat uses only `ansichat/config.toml` by default. It does not silently
reuse `copperline/config.toml` or Copperline paths for logs, ignore rules,
scripts, and relay keys. Copy an existing configuration and SSH keys to the
AnsiChat directory before upgrading a relay; explicit paths still work.

The GUI/Android app ID remains `ca.cerberusgames.copperline` to retain private
settings and keys and allow Android updates with the same signing certificate.
The relay wire protocol remains version 2 and keeps its existing channel
identifier and default SSH username (`copperline`), allowing existing clients
and relays to connect. Custom CTCP VERSION replies remain unchanged; the built-in
reply and `/flex` now identify AnsiChat.

New Lua scripts use `ansichat.*`; `copperline.*` remains an alias for existing
scripts. No script conversion is required.

For the complete TOML reference, defaults, theme options, DCC, SASL, IRCv3 capability settings, keybindings, logging, and relay settings, see [`CONFIGURATION.md`](CONFIGURATION.md).

For Lua scripts, custom commands, event hooks, and the complete scripting API, see [`SCRIPTING.md`](SCRIPTING.md).

For the native SSH relay/bouncer mode, including server/client TOML examples and key setup, see [`RELAY.md`](RELAY.md).

AnsiChat defaults to:

```text
~/.config/ansichat/config.toml
```

or `$XDG_CONFIG_HOME/ansichat/config.toml` when `XDG_CONFIG_HOME` is set.

Start with `config.example.toml`:

```sh
mkdir -p ~/.config/ansichat
cp config.example.toml ~/.config/ansichat/config.toml
```

A different configuration file can be selected with:

```sh
./ansichat -config=./my-config.toml
```

### Passwords and secrets

For passwords and tokens, environment variables are preferable to storing secrets directly in TOML. AnsiChat reads the variable named by `password_env`, `token_env`, or the relevant secret setting.

For example, a Libera SASL configuration can use:

```toml
[[server]]
name = "libera"
host = "irc.libera.chat"
tls = true

[server.sasl]
mechanism = "plain"
username = "myNick"
password_env = "LIBERA_IRC_PASSWORD"
```

With fish:

```fish
read -s -P 'Libera IRC password: ' LIBERA_IRC_PASSWORD
set -x LIBERA_IRC_PASSWORD $LIBERA_IRC_PASSWORD
./ansichat
```

For bash or zsh:

```sh
export LIBERA_IRC_PASSWORD='your-password-here'
./ansichat
```

### Message date and time

To show the date as well as the time, set `timestamp` in the existing `[general]` section of that machine's AnsiChat TOML configuration:

```toml
[general]
timestamp = "2006-01-02 15:04:05"
```

Restart that AnsiChat client after saving. Messages display in the machine's local time, for example `2026-09-11 08:30:45`. Relay clients format their own timestamps, so you can enable dates on one machine while leaving the others unchanged; no relay-server configuration change is needed.

The setting uses Go's reference-time layout: keep the literal reference date `2006-01-02` in the configuration rather than substituting today's date.

| Layout | Display |
| --- | --- |
| `15:04` | Hours and minutes (default) |
| `15:04:05` | Hours, minutes, and seconds |
| `2006-01-02 15:04` | Date, hours, and minutes |
| `2006-01-02 15:04:05` | Date and time with seconds |

`[general].timestamp` controls the format used for displayed messages and newly written logs. `[theme].timestamp` controls the timestamp color. See [the configuration reference](CONFIGURATION.md#timestamp-format) for more configuration details.

### Native SSH relay mode

Select the AnsiChat role in TOML:

```toml
[relay]
mode = "direct" # direct, server, or client
```

A relay server contains the normal `[[server]]` IRC definitions. A relay client may omit them completely because it receives IRC network and channel state from the relay.

Relay-client mode has a dedicated always-visible reconnect control. By default:

```toml
[keybindings]
relay_reconnect = "Alt+R"
```

Force reconnect tears down only the client's AnsiChat SSH attachment. It does not tell the relay to disconnect from IRC.

See [`RELAY.md`](RELAY.md) for host-key fingerprint setup, generated client keys, `authorized_keys`, server/client configuration, retained history, and security notes.

### Logging

Logging is enabled by default. AnsiChat can show a small persisted conversation preview from before the current process/session while keeping current-session messages in normal live styling.

Disable all on-disk IRC logs while keeping normal in-memory scrollback with:

```toml
[general]
logging = false
```

The default log location is:

```text
~/.local/state/ansichat/logs/<server>/<buffer>.log
```

Log files are created with mode `0600`; directories use `0700`.

### Gotify notifications

AnsiChat can push incoming mentions and private messages to Gotify. Gotify delivery is asynchronous so a slow or unavailable notification service does not freeze IRC or the TUI. When configured on the relay server, alerts continue with clients attached or detached. Use `/gotify status` on an updated relay client to inspect the relay's delivery counters and last error; `/gotify test` sends a test from the relay. In direct mode these commands inspect the local notifier.

```toml
[gotify]
enabled = true
url = "https://push.example.com"
token_env = "ANSICHAT_GOTIFY_TOKEN"
mentions = true
private_messages = true
priority = 5
timeout_seconds = 5
```

Test it from AnsiChat with:

```text
/gotify test
```

### Theme

AnsiChat ships with a dark navy/copper/aqua theme, but the entire TUI palette can be overridden with `[theme]`.

```toml
[theme]
background = "#090d16"
panel = "#101827"
foreground = "#d8dee9"
muted = "#77839a"
border = "#36506b"
title = "#64d8cb"
selected_fg = "#071018"
selected_bg = "#d99058"
timestamp = "#77839a"
server = "#64d8cb"
channel = "#8bd49c"
query = "#d9a7ff"
unread = "#f6c177"
mention = "#ff9ecb"
notice = "#f6c177"
action = "#d9a7ff"
system = "#72c7ef"
error = "#ff6b81"
dcc = "#8bd49c"
topic = "#f2cc8f"
input = "#eef1f6"
cursor_fg = "#071018"
cursor_bg = "#64d8cb"
status_fg = "#071018"
status_bg = "#64d8cb"
nick_colors = ["#ff7b72", "#f6c177", "#8bd49c", "#64d8cb", "#72c7ef", "#d9a7ff"]
```

Nick colors are chosen deterministically from `nick_colors`, so the same nickname keeps the same color.
