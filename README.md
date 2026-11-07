# AnsiChat

AnsiChat is a multi-network IRC client written in Go, with a terminal interface,
a desktop and Android GUI, and a built-in SSH relay/bouncer. Connect directly to
IRC, or keep your sessions running on a relay and attach from multiple devices.

Formerly **Copperline**. See the [upgrade guide](SETUP.md#upgrading-from-copperline)
if you have an existing installation.

[Quick start](#quick-start) · [Configuration](CONFIGURATION.md) ·
[Commands and controls](USAGE.md) · [SSH relay](RELAY.md) ·
[Desktop and Android](GUI.md) · [Release notes](CHANGELOG.md)

![AnsiChat terminal interface](screenshots/screenshot2.avif)

## Quick start

### 1. Build the terminal client

Install **Go 1.26 or newer**, then run from the repository root:

```sh
go build -o ansichat ./cmd/ansichat
./ansichat -version
```

For the desktop GUI or Android APK, follow the [GUI guide](GUI.md).

### 2. Create a basic configuration

Save this as `my-config.toml` and replace `YourNick` with your IRC nickname:

```toml
[general]
nick = "YourNick"
user = "YourNick"
real_name = "AnsiChat User"
timestamp = "15:04"

[[server]]
name = "libera"
host = "irc.libera.chat"
port = 6697
tls = true
auto_connect = false
```

Direct mode currently requires at least one `[[server]]` entry at startup.
This profile keeps auto-connect off so you can choose a server with `/quick`.
Set `auto_connect = true` if you prefer to connect to this configured network
when AnsiChat starts.

Launch with your profile:

```sh
./ansichat -config=./my-config.toml
```

Without `-config`, AnsiChat reads `~/.config/ansichat/config.toml`, or
`$XDG_CONFIG_HOME/ansichat/config.toml` when `XDG_CONFIG_HOME` is set.
See [config.example.toml](config.example.toml) for a fuller example and the
[configuration reference](CONFIGURATION.md) for all settings.

### 3. Connect with /quick

Inside AnsiChat, enter:

```text
/quick irc.libera.chat
```

`/quick` connects to a server without adding it to your configuration and opens
its status buffer. It uses your `[general]` identity and defaults to **TLS on
port 6697**, with certificate verification.

To specify a port and nickname, or use a plaintext connection:

```text
/quick irc.example.org 6697 OtherNick
/quick irc.example.org:6697 OtherNick
/quick -plain irc.example.org 6667
```

Replace `irc.example.org` with the server you want to use. Temporary sessions
last until AnsiChat exits; `/quick` does not save them to TOML or inherit a
configured network's passwords, SASL settings, client certificates, or channels.
For a network that needs authentication, use a
[configured server](CONFIGURATION.md) instead.

In **relay mode**, `/quick` makes the IRC connection on the relay and shares it
with attached clients. Both the client and relay need **0.2.37 or newer**;
temporary networks disappear when the relay restarts.

### 4. Join a channel and chat

Once connected, join a channel of your choice:

```text
/join #your-channel
```

Type a message and press **Enter** to send. Other useful commands:

| Command | Action |
| --- | --- |
| `/query Alice` | Open a private conversation |
| `/nick NewNick` | Change your nickname |
| `/part` | Leave the current channel |
| `/server` | List known servers |
| `/server libera` | Switch to a server's status buffer |
| `/connect libera` | Connect the configured network named `libera` |
| `/help` | List terminal commands |
| `/quit` | Exit AnsiChat |

`/server` selects a buffer; `/connect` connects an existing server entry.
Use `/quick` to create a temporary connection to an unconfigured server.

Use **Ctrl+N / Ctrl+P** to switch buffers, **Alt+A** for the next unread buffer,
**Tab** to complete a nickname, and **PageUp / PageDown** to scroll.
See the [user guide](USAGE.md) for all commands and controls.

## What AnsiChat includes

- Multiple IRC networks, channel and private-query buffers, drafts, search,
  unread markers, and mention highlighting.
- An embedded SSH relay with public-key authentication, pinned host keys,
  retained history, multiple attached clients, and reconnect controls.
- TLS, SASL PLAIN/EXTERNAL, IRCv3 capabilities, typing indicators, and history
  requests where supported by the network.
- Lua scripts, Gotify notifications, persistent logs, DCC transfers, configurable
  keybindings, and true-color themes.
- A shared Fyne GUI for desktop and Android that connects through an AnsiChat relay.

Read the [feature guide](FEATURES.md) for details.

## Documentation

| Guide | Contents |
| --- | --- |
| [Setup and upgrades](SETUP.md) | Building, configuration examples, secrets, logging, themes, and migration from Copperline |
| [Configuration reference](CONFIGURATION.md) | TOML options and defaults |
| [User guide](USAGE.md) | Terminal layout, keyboard controls, slash commands, and `/quick` |
| [SSH relay](RELAY.md) | Persistent sessions, server/client setup, keys, and history |
| [Desktop and Android](GUI.md) | GUI setup, desktop builds, and Android packaging |
| [Lua scripting](SCRIPTING.md) | Custom commands, events, and the scripting API; [example scripts](scripts/) |
| [Features](FEATURES.md) | Relay behavior, IRCv3 capabilities, DCC, and mention highlighting |
| [Release notes](CHANGELOG.md) | Changes by version |
| [Contributing](CONTRIBUTING.md) | Code policy, validation, and project layout |

## License

AnsiChat is licensed under the **GNU General Public License version 3 or later
(GPL-3.0-or-later)**. See [LICENSE](LICENSE).
