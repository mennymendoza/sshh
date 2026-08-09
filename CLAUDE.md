# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`sshh` — a tiny SSH chat server, client TUI, and history-fetching CLI, all in one Go binary (module `github.com/mennymendoza/sshh`).

## Commands

```bash
go build ./...              # build everything
go run ./cmd/sshh server    # run the chat server (see flags below)
go run ./cmd/sshh client    # run the terminal chat client
go run ./cmd/sshh history   # fetch/decrypt stored history (paginated)
go vet ./...
gofmt -l .                  # there are no *_test.go files yet
```

Generating an X25519 keypair for message-at-rest encryption (there is no built-in keygen command):

```bash
openssl genpkey -algorithm X25519 -out sshh.key
openssl pkey -in sshh.key -pubout -out sshh.pub
```

Server usage:

```bash
go run ./cmd/sshh server --addr :2222 --host-key host_key [--db sshh.db --pub sshh.pub]
go run ./cmd/sshh client --addr localhost:2222 --user alice --room general
go run ./cmd/sshh client --addr localhost:2222 --user alice --room general --message "hello"   # one-shot send, no session
go run ./cmd/sshh client --addr localhost:2222 --user alice --room general --stream            # print incoming messages, no TUI
go run ./cmd/sshh history --addr localhost:2222 --key sshh.key --room general [--user alice] [--json] [--page 1] [--pagesize 100]
```

`--db` and `--pub` must be supplied together (`MarkFlagsRequiredTogether`); without them the server runs in-memory only with no persistence and `history` has nothing to fetch.

## Architecture

Everything rides on **one SSH channel type, `"chat"`**, carrying newline-delimited JSON. There is no real SSH authentication — `PublicKeyCallback` and `PasswordCallback` in `internal/sshserver/server.go` both unconditionally succeed, so any key or password is accepted; SSH here is purely a transport with a friendly client (`ssh user@host -p 2222` also works against it), not an auth boundary.

- **`cmd/sshh`** — cobra root command wiring three subcommands (`server`, `client`, `history`) to their respective internal packages. Each subcommand file (`server.go`, `client.go`, `history.go`) only parses flags and calls `Run(...)` in the matching internal package.
- **`internal/protocol`** — the wire contract shared by server and both clients: `ClientMessage` (`join`, `send`, `list_rooms`, `list_users`, `history`; `join` accepts a `Quiet` flag to suppress `user_joined`/`user_left` broadcasts; `history` accepts `Page`/`PageSize`, both 1-indexed and defaulted server-side to `1`/`100` when zero) and `ServerMessage` (`ack`, `error`, `message`, `rooms`, `users`, `user_joined`, `user_left`, `history`; the `history` result echoes back the resolved `Page`/`PageSize`). Changing message shapes here affects `sshserver`, `tui`, and `historyclient` simultaneously.
- **`internal/sshserver`** — the server. `server.go` accepts SSH connections and rejects any channel that isn't `"chat"`. `session.go` is the core state machine: one `session` per channel, with a `writeLoop` goroutine draining an `out` channel to the SSH channel, and a `relayLoop` goroutine forwarding room broadcasts into `out` after `join`. `hostkey.go` auto-generates and persists an ed25519 host key on first run if the configured path doesn't exist.
- **`internal/room`** — in-memory pub/sub only (`Registry`), keyed by room name, mapping each subscriber channel to the username that owns it (also used to answer `list_users`); not persisted and rebuilt on every server restart. This is what makes live broadcast work regardless of whether SQLite persistence is enabled.
- **`internal/db`** — optional SQLite persistence (via `modernc.org/sqlite`, no cgo) for rooms/messages, schema embedded from `schema.sql` via `go:embed` and applied idempotently on `Open`. Only wired up when the server is started with both `--db` and `--pub`. `ListMessages(roomID, page, pageSize)` paginates with `LIMIT`/`OFFSET`, ordered by `created_at`; `page` is 1-indexed.
- **`internal/cryptox`** — encrypts message bodies at rest using anonymous X25519 sealed boxes (`nacl/box.SealAnonymous`/`OpenAnonymous`), so the server can store ciphertext without holding the decryption key. Keys are hand-parsed from PKIX (public) / PKCS8 (private) PEM — the format `openssl genpkey -algorithm X25519` produces — since Go's stdlib doesn't natively expose X25519 key parsing.
- **`internal/historyclient`** — a headless SSH client that dials in, sends a `history` request (with `Page`/`PageSize`) over a `chat` channel, and decrypts the returned ciphertext locally with the operator's private key (never sent to the server). The `--user` filter is applied client-side to whatever page comes back, after decryption.
- **`internal/tui`** — the chat client, built on `bubbletea`/`bubbles`/`lipgloss`. `client.go` holds three entry points: `Run` (interactive TUI; `readLoop` forwards SSH channel messages to `tea.Program.Send`, and the bubbletea `model` renders them), `RunOnce` (headless one-shot send: joins quietly via `ClientMessage.Quiet` so no `user_joined`/`user_left` broadcast fires, sends one message, waits for it to echo back, exits), and `RunStream` (headless: joins normally, then prints every incoming `message` event to stdout as `sender: body` with no TUI). `style.go` holds all lipgloss styling used by `Run`. Slash commands (`/rooms`, `/users`, `/join <room>`, `/clear`, `/help`) are handled inline in `Update` by writing `protocol.ClientMessage`s back over the channel rather than being a separate command layer.

### Data flow for a chat message

`tui` (or any SSH client) writes a `ClientMessage{Type: send}` on its `chat` channel → `session.handleSend` optionally encrypts + persists it via `db`/`cryptox` → broadcasts the plaintext `ServerMessage{Type: message}` to every subscriber in the room via `room.Registry.Broadcast` → each session's `relayLoop` forwards it into that session's `out` channel → `writeLoop` writes it back down the SSH channel.
