<div align="center">

```
██╗    ██╗███████╗
██║    ██║██╔════╝
██║ █╗ ██║███████╗
██║███╗██║╚════██║
╚███╔███╔╝███████║
 ╚══╝╚══╝ ╚══════╝
```

**Fast, git-aware workspace manager for your terminal.**

[Install](#install) · [Usage](#usage)

</div>

---

## Why `ws`?

Your shell already has `cd`. But `cd ~/work/client-a/backend/services/api` isn't fun, and `cd`-ing into the right git worktree for a feature branch is worse. `ws` turns that into:

```sh
ws api            # picker if there are worktrees, otherwise jumps
ws api/           # jumps straight to the root
ws api/feat-x     # jumps straight to the worktree you aliased
```

### Highlights

- **Instant jumps** — `ws <name>` resolves exact → prefix → fuzzy, cold-starts in milliseconds, no daemon
- **Worktree-aware** — one workspace owns a git root and every worktree git reports; dedup is automatic when you add either
- **Smart add wizard** — `ws add` walks the directory tree, finds git projects (including nested), shows them in a multi-select with auto-detected project icons, lets you alias each root and (optionally) each worktree in a single-screen form
- **Grouped TUI picker** — groupable workspaces, fuzzy search, expand/collapse worktrees, Nerd Font icons per project type (node, next, rust, go, python, ios, android, tauri, electron, ruby, java, docker…)
- **Empty-state onboarding** — first `ws` shows a welcome screen with a single keypress to add the current directory
- **Single binary, plain TOML** — config at `~/.config/ws/config.toml`, hand-editable, dotfiles-friendly
- **Works with bash, zsh, fish** — `ws init <shell>` prints the wrapper function that does the actual `cd`

## Install

`ws` ships as a single static binary. There are three ways to get it — pick one.

### 1. One-liner (release binary)

```sh
curl -fsSL https://raw.githubusercontent.com/sonla58/ws-cli/main/scripts/install.sh | bash
```

Installs into `$HOME/.local/bin`. Override the prefix with `PREFIX=/usr/local`
or pin a version with `WS_VERSION=v0.1.0`.

### 2. `go install`

```sh
go install github.com/sonla58/ws-cli/cmd/ws@latest
```

Puts the binary in `$(go env GOPATH)/bin`.

### 3. From source

```sh
git clone https://github.com/sonla58/ws-cli
cd ws-cli
make install              # installs to $HOME/.local/bin
# or: PREFIX=/usr/local sudo make install
```

### Finish the install — shell integration

A child process can't change your parent shell's directory, so `ws` prints
the chosen path and a one-line shell function `cd`s for you. Add to your rc
file (pick one):

```sh
# zsh (~/.zshrc)
eval "$(ws init zsh)"

# bash (~/.bashrc)
eval "$(ws init bash)"

# fish (~/.config/fish/config.fish)
ws init fish | source
```

Open a new terminal (or `source` your rc file). Verify with:

```sh
type ws     # → "ws is a shell function"  (not "ws is /path/to/ws")
ws add      # → wizard opens in your current directory
```

> **Common gotcha:** if calling `ws api` prints the path but doesn't `cd`,
> the shell function isn't loaded. Run `type ws` — if it reports a file
> path instead of "shell function", you're invoking the binary directly.
> Make sure the `eval "$(ws init ...)"` line is in your rc file AND the rc
> file has been sourced in the current shell.

## Usage

```sh
ws                   # TUI picker (onboarding screen if empty)
ws <name>            # jump directly (opens a narrowed picker if the workspace has worktrees)
ws <name>/           # jump straight to the workspace root, skipping the picker
ws <name>/<suffix>   # jump to an explicitly-aliased worktree
ws <group>           # open picker filtered to a group
ws add [path]        # wizard: find projects under path, let you choose + name them
ws rm <name>         # remove a workspace or worktree
ws list              # plain-text listing (grouped)
ws refresh [name]    # re-detect icons and pick up new worktrees
ws pick              # open picker explicitly (same as bare `ws`)
ws init <shell>      # print shell integration script
```

### Picker keybindings

| key | action |
| --- | --- |
| `↑` `↓` / `j` `k` | move |
| `⏎` | open / expand / pick root |
| `→` `←` / `l` `h` | expand / collapse worktrees |
| `/` | fuzzy search |
| `a` | add current dir |
| `q` `esc` | quit |

### Add-wizard keybindings

**Select step**

| key | action |
| --- | --- |
| `space` | toggle selection on focused row |
| `a` | select all · `n` select none |
| `⏎` | confirm selection → naming |
| `esc` | cancel |

**Name step** (single-screen form)

| key | action |
| --- | --- |
| `↑` `↓` / `Tab` `⇧Tab` | move between rows |
| `⏎` | next row · submit from last row |
| `a` | on a worktree row: enable aliasing (reveals `${root}/` prefix + editable suffix) |
| `ctrl+s` | submit from any row |
| `esc` | back to select step · `ctrl+c` cancel |

**Group step**

| key | action |
| --- | --- |
| typing | edit group name (leave empty for ungrouped) |
| `⏎` | confirm and save |
| `ctrl+c` `esc` | cancel |

## Configuration

Config lives at `$XDG_CONFIG_HOME/ws/config.toml` (usually `~/.config/ws/config.toml`).
Override with `$WS_CONFIG`. It's human-editable TOML — hand-edit freely or commit
it to a dotfiles repo.

```toml
schema = 1

[scan]
depth = 2
ignores = ["node_modules", ".venv", "target", "dist"]

[[groups]]
name = "work"

[[workspaces]]
name  = "api"
path  = "/Users/you/code/api"
group = "work"
icon  = "node"

  [[workspaces.worktrees]]
  name = "api-feat"
  path = "/Users/you/code/api-feat"
  icon = "node"
```

Set `NO_NERD_FONT=1` to fall back to plain-letter icons if your font lacks the
Nerd Font glyphs.

## How worktree workspaces work

`ws` knows about `git worktree list --porcelain`. One workspace represents a repo
and every worktree that repo currently has.

**Adding:**

- `ws add` on a git **root** → saves the root and attaches every worktree
- `ws add` on a **worktree path** → `ws` resolves the common root. If the root
  is already saved, only the new worktree is attached (deduped). If not, the
  root is saved normally — which pulls in all worktrees in one shot.

**Aliasing worktrees is optional.** During the add wizard, worktree rows start
unaliased; press `a` on a row to enable aliasing (the UI shows a read-only
`${root}/` prefix followed by an editable suffix). Unaliased worktrees are
still saved — they live under the root in the picker and are reachable by
expanding it.

**Jumping:**

| command | behavior |
| --- | --- |
| `ws api` | picker, narrowed to `api` + its worktrees (root is selectable as `(root)`) |
| `ws api/` | straight to the **root** path (skip the picker) |
| `ws api/feat-x` | straight to the explicitly-aliased worktree |
| bare `ws` / `ws <group>` | full picker (reaches unaliased worktrees) |

**Refreshing:** `ws refresh [name]` re-runs `git worktree list` and pulls new
worktrees in unaliased (so they don't collide with your naming scheme).

## Project-type icons

`ws` detects the type of each workspace from signature files at add / refresh
time and caches it in the config. Detected types:

| type | signatures |
| --- | --- |
| node | `package.json` |
| nextjs | `next.config.{js,mjs,ts}` |
| tauri | `tauri.conf.json`, `src-tauri/` |
| electron | `electron-builder.*`, `electron.vite.config.ts` |
| rust | `Cargo.toml` |
| go | `go.mod` |
| python | `pyproject.toml`, `requirements.txt`, `setup.py`, `Pipfile` |
| ios | `Podfile`, `*.xcodeproj`, `*.xcworkspace` |
| android | `AndroidManifest.xml`, `build.gradle*`, `settings.gradle` |
| ruby | `Gemfile` |
| java | `pom.xml` |
| docker | `Dockerfile`, `docker-compose.yml`, `compose.yml` |

Icons render via Nerd Font glyphs by default. Set `NO_NERD_FONT=1` to fall
back to plain letters.
