# ws

Fast, git-aware workspace manager for your terminal.

- **Jump instantly**: `ws <name>` cd's to a saved workspace
- **Browse in a TUI**: bare `ws` opens an elegant grouped picker
- **Worktree-aware**: one workspace represents a git root and all its worktrees
- **Smart add**: `ws add` walks the current tree, finds git projects (and their worktrees), and lets you pick & name them
- **Single binary, single TOML**: fast cold start; no daemon, no database

## Install

`ws` ships as a single static binary. There are three ways to get it — pick one.

### 1. One-liner (release binary)

```sh
curl -fsSL https://raw.githubusercontent.com/longnguyen/ws-cli/main/scripts/install.sh | bash
```

Installs into `$HOME/.local/bin`. Override the prefix with `PREFIX=/usr/local`
or pin a version with `WS_VERSION=v0.1.0`.

### 2. `go install`

```sh
go install github.com/longnguyen/ws-cli/cmd/ws@latest
```

Puts the binary in `$(go env GOPATH)/bin`.

### 3. From source

```sh
git clone https://github.com/longnguyen/ws-cli
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

```
↑/↓     move          ⏎   open / expand
→/←     expand/collapse    /   fuzzy search
a       add current dir    q/esc   quit
```

### Add-wizard keybindings

```
space   toggle selection    a   select all
n       select none         ⏎   next step / confirm
esc     back / cancel
```

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

- `ws add` on a git **root**: saves the root and attaches all its worktrees.
- `ws add` on a **worktree**: resolves the common root via `git`. If the root is
  already saved, the worktree is attached (deduped). If not, the root is saved
  normally (which pulls in all its worktrees).
- `ws <alias>` on a workspace with **no extra worktrees**: cd's directly.
- `ws <alias>` on a workspace with **multiple worktrees**: opens the picker
  narrowed to that workspace with its worktrees expanded.

## Development

```sh
make build       # build binary
make test        # run tests
make install     # install to $GOPATH/bin
```
