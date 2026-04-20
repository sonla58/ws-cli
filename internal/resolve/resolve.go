package resolve

import (
	"strings"

	"github.com/longnguyen/ws-cli/internal/model"
	"github.com/sahilm/fuzzy"
)

// Hit is a resolved candidate.
type Hit struct {
	Workspace *model.Workspace
	// Worktree, if non-nil, means the query targeted a specific worktree entry.
	Worktree *model.Worktree
}

// Path of the hit — worktree path if present, otherwise workspace path.
func (h Hit) Path() string {
	if h.Worktree != nil {
		return h.Worktree.Path
	}
	return h.Workspace.Path
}

// Name of the hit.
func (h Hit) Name() string {
	if h.Worktree != nil {
		return h.Worktree.Name
	}
	return h.Workspace.Name
}

// Resolve finds candidate hits for query in cfg. Search order:
//  1. exact alias match (workspace name OR worktree name)
//  2. prefix match
//  3. fuzzy match (sahilm/fuzzy)
//
// Returns all hits at the best tier found (so the caller can decide: unique
// → jump, multiple → disambiguate in TUI).
func Resolve(cfg model.Config, query string) []Hit {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}

	// Flatten searchable entries, preserving workspace pointers.
	var all []Hit
	names := []string{}
	for i := range cfg.Workspaces {
		w := &cfg.Workspaces[i]
		all = append(all, Hit{Workspace: w})
		names = append(names, w.Name)
		for j := range w.Worktrees {
			wt := &w.Worktrees[j]
			// Empty-name worktrees aren't globally resolvable by query.
			if wt.Name == "" {
				continue
			}
			all = append(all, Hit{Workspace: w, Worktree: wt})
			names = append(names, wt.Name)
		}
	}

	// Tier 1: exact.
	var exact []Hit
	for i, n := range names {
		if n == q {
			exact = append(exact, all[i])
		}
	}
	if len(exact) > 0 {
		return exact
	}

	// Tier 2: prefix.
	lq := strings.ToLower(q)
	var prefix []Hit
	for i, n := range names {
		if strings.HasPrefix(strings.ToLower(n), lq) {
			prefix = append(prefix, all[i])
		}
	}
	if len(prefix) > 0 {
		return prefix
	}

	// Tier 3: fuzzy.
	matches := fuzzy.Find(q, names)
	var out []Hit
	for _, m := range matches {
		out = append(out, all[m.Index])
	}
	return out
}
