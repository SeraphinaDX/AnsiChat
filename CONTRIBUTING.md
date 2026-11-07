# Contributing to AnsiChat

[Back to README](README.md)

## LLM Code Policy

This project does not discriminate against the use of LLM generated code. The project already does contain LLM generated code. Just make sure the code compiles and does not introduce new bugs or cause it not to pass tests.

This project also chose the language Go precisely for its memory safety because of those guardrails for LLM generated code.

This code is daily driven by the author. All bugs are eliminated in a prompt manner by someone terminally online.

## Build and validation

Use Go 1.26 or newer. Build the terminal client with `make tui`. See the
[GUI guide](GUI.md) for desktop and Android build prerequisites.

```sh
go test -race -tags ci ./...
```

The `ci` tag selects Fyne's headless driver for testing. For application fixes
and features, follow the versioning convention in [AGENTS.md](AGENTS.md).

## Project layout

```text
cmd/ansichat/       program entry point
internal/config/      TOML configuration
internal/model/       buffers and messages
internal/irc/         direct multi-server IRC + IRCv3 backend
internal/relay/       embedded SSH relay server/client and protocol
internal/logging/     persistent logs and pre-session backlog
internal/gotify/      Gotify notification client
internal/dcc/         DCC parsing and transfers
internal/scripting/   embedded Lua runtime and scripting API
scripts/              example Lua scripts
internal/tui/         gotui interface, navigation, input, and commands
```
