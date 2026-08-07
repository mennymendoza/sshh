# sshh
a tiny SSH chat server. Pronounced "shush".

## Installation

```bash
go install github.com/mennymendoza/sshh/cmd/sshh@latest
```

Or build from source:

```bash
git clone https://github.com/mennymendoza/sshh.git
cd sshh
go build ./cmd/sshh
```

## Usage

Run the server:

```bash
sshh server --addr :2222
```

Connect with the client:

```bash
sshh client --addr localhost:2222 --user alice --room general
```

In the client, type a message and hit enter to send. Slash commands: `/rooms` (list rooms), `/users` (list users in the current room), `/join <room>` (switch rooms), `/clear` (clear the screen), `/help` (show available commands).

Send a single message without opening a session:

```bash
sshh client --addr localhost:2222 --user alice --room general --message "hello"
```

Print incoming messages to stdout without opening the TUI:

```bash
sshh client --addr localhost:2222 --user alice --room general --stream
```

### Persisting and decrypting history

To store messages (encrypted at rest), generate an X25519 keypair and start the server with `--db` and `--pub`:

```bash
openssl genpkey -algorithm X25519 -out sshh.key
openssl pkey -in sshh.key -pubout -out sshh.pub

sshh server --addr :2222 --db sshh.db --pub sshh.pub
```

Fetch and decrypt a room's history with the matching private key:

```bash
sshh history --addr localhost:2222 --key sshh.key --room general
```
