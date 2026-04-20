package scan

import (
	"io/fs"
	"os"
	"path/filepath"
)

// RepoHits are discovered git repos (directories containing `.git`).
type RepoHits struct {
	Root string // absolute path to repo root
}

// FindRepos walks `root` up to `maxDepth` levels (root=0). When a directory
// contains `.git`, it's reported and NOT descended into. Directories whose
// base name is in `ignores` are skipped entirely.
func FindRepos(root string, maxDepth int, ignores []string) ([]RepoHits, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	ig := make(map[string]struct{}, len(ignores))
	for _, s := range ignores {
		ig[s] = struct{}{}
	}

	// If root itself is a repo, return just it.
	if isRepoDir(abs) {
		return []RepoHits{{Root: abs}}, nil
	}

	var hits []RepoHits
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if path != abs {
			if _, skip := ig[base]; skip {
				return filepath.SkipDir
			}
		}
		depth := relDepth(abs, path)
		if depth > maxDepth {
			return filepath.SkipDir
		}
		if isRepoDir(path) {
			hits = append(hits, RepoHits{Root: path})
			return filepath.SkipDir
		}
		return nil
	})
	return hits, err
}

func isRepoDir(p string) bool {
	fi, err := os.Stat(filepath.Join(p, ".git"))
	return err == nil && (fi.IsDir() || fi.Mode().IsRegular()) // .git can be a file in worktrees
}

func relDepth(root, p string) int {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." {
		return 0
	}
	n := 0
	for _, r := range rel {
		if r == filepath.Separator {
			n++
		}
	}
	return n + 1
}
