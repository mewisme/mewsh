# MewSH

SSH profile manager with a terminal UI and CLI. Save hosts, users, and auth settings in JSON; store passwords in the OS keyring; launch the **native OpenSSH client** in a separate terminal window.

## Features

- **TUI** — profile list, filter, connect, add/edit/delete via menu
- **CLI** — scriptable `add`, `list`, `edit`, `delete`, `connect`, `doctor`
- **Direct SSH** — spawn `ssh` to host:port
- **Cloudflare Access** — shared `cloudflared` tunnel per hostname, multiple SSH sessions per tunnel
- **Background connect** — headless/scriptable SSH (`-b`) with log files and session tracking
- **Session manager** — list, kill, and kill-all (TUI and CLI)
- **Passwords** — OS keyring (`mewsh` service); never stored in `config.json`
- **Auto password** (Linux/macOS) — `expect` or `sshpass` when configured

## Install

### Homebrew (macOS / Linux)

```bash
brew tap mewisme/mewsh https://github.com/mewisme/mewsh
brew install --cask mewsh
```

After the first tagged release, GoReleaser updates `Casks/mewsh.rb` in this repo automatically.

If you previously installed the old formula, `brew upgrade mewsh` migrates to the cask via `tap_migrations.json`.

### Releases

Download a binary from [GitHub Releases](https://github.com/mewisme/mewsh/releases), or install with Go:

```bash
go install github.com/mewisme/mewsh@latest
```

### Build from source

```bash
git clone https://github.com/mewisme/mewsh.git
cd mewsh
go build -ldflags="-s -w" -o mewsh .
```

On Windows with Go 1.26+, use `-ldflags="-s -w"` so the executable runs correctly.

The default `mewsh` command opens the TUI — run it from a terminal (Windows Terminal, PowerShell, or cmd), not by double-clicking the binary.

## Quick start

```bash
# Interactive TUI (default)
mewsh

# Or use the CLI
mewsh add
mewsh list
mewsh connect myserver
```

## Configuration

| Platform | Config file |
|----------|-------------|
| Linux / macOS | `~/.config/mewsh/config.json` |
| Windows | `%APPDATA%\mewsh\config.json` |

Override path:

```bash
mewsh --config /path/to/config.json
```

| Path | Purpose |
|------|---------|
| `<config_dir>/config.json` | Profiles and settings |
| `<config_dir>/sessions/*.log` | Background SSH session output |
| `<config_dir>/bin/cloudflared(.exe)` | Bundled cloudflared (after `mewsh cloudflared update`) |

Config directory is created with restrictive permissions on Unix. Passwords use the system keyring only.

## Commands

```bash
mewsh                  # TUI (default)
mewsh --help           # CLI help

mewsh add              # Add profile (interactive)
mewsh edit <alias>     # Edit profile
mewsh delete <alias>   # Delete profile
mewsh list             # List profiles (table)
mewsh connect <alias>              # Connect (blocking; opens terminal)
mewsh connect <alias> -b           # Background (no GUI; survives shell exit)
mewsh sessions                     # List active SSH sessions (same as sessions list)
mewsh sessions list                # List sessions (alias: ls)
mewsh sessions list --json         # List sessions as JSON
mewsh sessions kill <id> [id...]   # Stop one or more sessions
mewsh sessions kill --alias <a>    # Stop all sessions for a profile
mewsh sessions kill --all          # Stop every session
mewsh doctor                       # Check ssh, terminal, cloudflared
mewsh cloudflared update           # Download/update bundled cloudflared
mewsh version                      # Show version, install method, release status
mewsh update                       # Update to latest release
mewsh update --check               # Only check for updates
mewsh update --force               # Reinstall even if up to date (go install: -a, GOPROXY=direct)
```

Global flag: `--config <path>` — custom config file location.

### Self-update

`mewsh update` detects how you installed mewsh and picks the right path:

| Install method | Update action |
|----------------|---------------|
| **Homebrew** | `brew upgrade mewsh` |
| **go install** | `go install github.com/mewisme/mewsh@<release-tag>` (latest GitHub tag) |
| **Binary** (release download) | Downloads from GitHub Releases and replaces the executable |

`mewsh version` prints build info, install method, and whether a newer release is available. Use `mewsh update --force` to reinstall via go install when already on the latest tag (bypasses module/build cache).

## TUI

```bash
mewsh
```

Press `?` anywhere for a multi-page help guide (profiles, menu, sessions, forms, confirmations). The footer shows key hints; profile count and active session count appear in the status line.

### Profile list

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Move selection |
| `/` | Filter by alias, host, user, or note |
| `Enter` | Connect (detached SSH in new terminal) |
| `m` | Open menu |
| `?` | Help guide |
| `q`, `Ctrl+C` | Quit (confirmation; default Yes) |
| `Esc` `Esc` | Quit quickly (double Esc) |

### Menu (`m`)

| Key | Action |
|-----|--------|
| `a` | Add profile |
| `e` | Edit selected profile |
| `d` | Delete selected profile (confirm; default No) |
| `s` | Active sessions |
| `m` | Close menu |

Footer shows live **Active sessions** count when SSH sessions are running.

### Sessions (`m` → `s`)

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Move selection |
| `/` | Filter by alias or target |
| `Space` | Mark / unmark session |
| `Enter` | Kill marked session(s), or selected if none marked (confirm) |
| `a` | Kill all listed sessions (confirm) |
| `m` / `Esc` | Back to profiles |
| `?` | Help guide |

## Profile fields

| Field | Description |
|-------|-------------|
| `alias` | Unique name (required) |
| `connection_type` | `direct` or `cloudflare_access` |
| `host` | Required for direct SSH |
| `port` | Default `22` |
| `user` | SSH username (required) |
| `auth_type` | `agent`, `key`, or `password` |
| `key_path` | Required when `auth_type` is `key` |
| `password_mode` | `manual` (default) or `auto` |
| `password_ref` | Keyring key (defaults to alias) |
| `cf_hostname` | Required for Cloudflare Access |
| `note` | Optional note |

## Authentication

| Type | Behavior |
|------|----------|
| `agent` | `ssh user@host` (SSH agent) |
| `key` | `ssh -i <key_path> user@host` |
| `password` + `manual` | Native `ssh`; password at prompt |
| `password` + `auto` | `expect`/`sshpass` (Unix) or keyring + `SSH_ASKPASS` (Windows) |

## Connect behavior

### Direct

Opens a new terminal running native `ssh` to `host:port`.

### Cloudflare Access

1. Resolve `cloudflared` (bundled → `PATH` → download via `mewsh cloudflared update`)
2. Start a local tunnel: `cloudflared access ssh --hostname <cf_hostname> --url 127.0.0.1:<port>`
3. Wait until the port is ready
4. Open SSH to `127.0.0.1:<local_port>` in a new terminal
5. **Shared tunnel** — multiple profiles/sessions to the same `cf_hostname` reuse one tunnel (ref-counted)
6. Tunnel stops only after the last SSH session ends

TUI connect uses **detached** mode (returns to the profile list immediately). CLI `mewsh connect` blocks until SSH exits.

### Background (`-b` / `--background`)

For headless Linux servers (no desktop) or scripts, start SSH without a terminal emulator:

```bash
mewsh connect myserver --background
```

This spawns a detached **mewsh worker** (`__bg-connect__`) in a new session (fully detached from your terminal on Windows). The worker keeps the Cloudflare tunnel alive (if needed), runs `ssh` without allocating a local TTY, and writes output to `<config_dir>/sessions/<alias>-<timestamp>.log`. Your shell returns immediately; the session keeps running after you log out of the server.

For **Cloudflare Access** profiles, the worker holds one `cloudflared` tunnel for the whole session (instead of `ProxyCommand` per SSH handshake), which avoids extra console flashes on Windows.

On start, mewsh prints the session id, worker PID, log path, and copy-paste **SSH command hints** (interactive terminal vs background worker). List or stop sessions with `mewsh sessions` and `mewsh sessions kill`.

## Terminal spawning

Spawned windows use the title **MewSH** where the platform allows it.

| Platform | Launchers (in order) |
|----------|----------------------|
| Windows | Batch helper → Windows Terminal (`wt`) → `cmd` |
| macOS | Terminal.app (`osascript`) |
| Linux | `$TERMINAL`, gnome-terminal, konsole, kitty, alacritty, wezterm, xterm, … |

If spawning fails, the full `ssh` command is printed so you can run it manually.

## Releases (maintainers)

Tag a version to trigger GoReleaser:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The [Release](.github/workflows/release.yml) workflow will:

- Build Linux, macOS, and Windows binaries (amd64, arm64, and 386 for Linux/Windows)
- Publish GitHub release archives and checksums
- Commit an updated Homebrew cask to `Casks/mewsh.rb` on `main`

[CI](.github/workflows/ci.yml) runs tests on push/PR.

**Note:** The release job needs permission to push to `main` (for the cask commit). The default `GITHUB_TOKEN` in the workflow is sufficient when the cask lives in this repository.

After the first cask release, remove any legacy `Formula/mewsh.rb` from the default branch if it exists.

## Requirements

- [OpenSSH](https://www.openssh.com/) client (`ssh` in `PATH`)
- For Cloudflare Access: [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/) (bundled download supported)
- For password auto mode on Unix: `expect` or `sshpass`

## Security

- Passwords only in the OS keyring, not in `config.json`
- Config file mode `0600` on Unix
- SSH/cloudflared invoked with argument slices where possible
- Alias uniqueness and port validation enforced

## License

MIT — see [LICENSE](LICENSE).
