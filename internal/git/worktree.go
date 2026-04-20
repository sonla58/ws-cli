package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Entry describes one worktree from `git worktree list --porcelain`.
type Entry struct {
	Path   string
	Branch string // refs/heads/<name> or empty for detached
	Bare   bool
}

// Available returns true if `git` is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsRepo returns true if dir is inside a git working tree.
func IsRepo(dir string) bool {
	if !Available() {
		return false
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// CommonRoot returns the common root (main worktree) for any worktree path.
// It uses `git rev-parse --git-common-dir` + `rev-parse --show-toplevel`
// executed against the common dir.
func CommonRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", err
	}
	commonDir := strings.TrimSpace(string(out))
	// commonDir is typically `<root>/.git`; resolve parent (but handle bare repos).
	if filepath.Base(commonDir) == ".git" {
		return filepath.Dir(commonDir), nil
	}
	// Fallback: toplevel.
	top, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(top)), nil
}

// Worktrees lists all worktrees for the repo containing dir. Returns entries
// with the main worktree first.
func Worktrees(dir string) ([]Entry, error) {
	out, err := exec.Command("git", "-C", dir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(string(out)), nil
}

func parseWorktreeList(s string) []Entry {
	var entries []Entry
	var cur Entry
	flush := func() {
		if cur.Path != "" {
			entries = append(entries, cur)
		}
		cur = Entry{}
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			cur.Bare = true
		}
	}
	flush()
	return entries
}
