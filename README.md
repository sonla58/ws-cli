# ws

Fast, git-aware workspace manager for your terminal.

- **Jump instantly**: `ws <name>` cd's to a saved workspace
- **Browse in a TUI**: bare `ws` opens an elegant grouped picker
- **Worktree-aware**: one workspace represents a git root and all its worktrees
- **Smart add**: `ws add` walks the current tree, finds git projects (and their worktrees), and lets you pick & name them
- **Single binary, single TOML**: fast cold start; no daemon, no database

## Install

```sh
go install github.com/longnguyen/ws-cli/cmd/ws@latest
```

Add shell integration to your rc file:

```sh
# zsh (~/.zshrc)
eval "$(ws init zsh)"

# bash (~/.bashrc)
eval "$(ws init bash)"

# fish (~/.config/fish/config.fish)
ws init fish | source
```

The shell function captures the chosen path from `ws` and `cd`s your live shell to it — something a child process can't do on its own.

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
