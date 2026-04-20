package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/longnguyen/ws-cli/internal/config"
	"github.com/longnguyen/ws-cli/internal/detect"
	"github.com/longnguyen/ws-cli/internal/git"
	"github.com/longnguyen/ws-cli/internal/model"
	"github.com/longnguyen/ws-cli/internal/resolve"
	"github.com/longnguyen/ws-cli/internal/scan"
	"github.com/longnguyen/ws-cli/internal/shell"
	"github.com/longnguyen/ws-cli/internal/tui"
)

var (
	version  = "0.1.0"
	emitPath bool
)

// out returns the stream user-facing messages should use. When --emit-path is
// set, stdout is reserved for the chosen path only, so everything else goes
// to stderr so the shell user still sees it.
func out() io.Writer {
	if emitPath {
		return os.Stderr
	}
	return os.Stdout
}

func main() {
	root := &cobra.Command{
		Use:     "ws [name]",
		Short:   "Fast workspace manager",
		Version: version,
		Args:    cobra.ArbitraryArgs,
		RunE:    runRoot,
	}
	root.PersistentFlags().BoolVar(&emitPath, "emit-path", false, "emit chosen path to stdout (used by shell wrapper)")

	root.AddCommand(cmdAdd(), cmdRemove(), cmdList(), cmdInit(), cmdRefresh(), cmdPick())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runRoot: bare `ws` → picker. `ws <name>` → resolve + maybe narrowed picker.
func runRoot(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return openPicker(cfg, nil)
	}
	q := strings.Join(args, " ")
	// Trailing slash means "the root of this workspace" — handy when a
	// workspace has worktrees and `ws <name>` would otherwise open the picker.
	rootOnly := false
	if strings.HasSuffix(q, "/") && len(q) > 1 {
		rootOnly = true
		q = strings.TrimSuffix(q, "/")
	}
	hits := resolve.Resolve(cfg, q)

	// Exact workspace/worktree name match wins outright.
	hasExactName := false
	for _, h := range hits {
		if h.Name() == q {
			hasExactName = true
			break
		}
	}

	// If no exact-name match, a group name match opens a group-filtered picker.
	if !hasExactName {
		if names, ok := workspacesInGroup(cfg, q); ok {
			if len(names) == 0 {
				fmt.Fprintf(os.Stderr, "ws: group %q has no workspaces\n", q)
				os.Exit(1)
			}
			return openPicker(cfg, names)
		}
	}

	if len(hits) == 0 {
		fmt.Fprintf(os.Stderr, "ws: no workspace or group matches %q\n", q)
		os.Exit(1)
	}
	if len(hits) == 1 {
		h := hits[0]
		if rootOnly && h.Workspace != nil {
			emitChosen(h.Workspace.Path)
			return nil
		}
		// Worktree hit or workspace without extra worktrees → jump.
		if h.Worktree != nil || len(h.Workspace.Worktrees) == 0 {
			emitChosen(h.Path())
			return nil
		}
		// Workspace has worktrees → open narrowed picker so user can choose
		// between root and worktrees. Use `ws <name>/` to jump straight to root.
		return openPicker(cfg, []string{h.Workspace.Name})
	}
	names := map[string]bool{}
	var uniq []string
	for _, h := range hits {
		if !names[h.Workspace.Name] {
			names[h.Workspace.Name] = true
			uniq = append(uniq, h.Workspace.Name)
		}
	}
	return openPicker(cfg, uniq)
}

// workspacesInGroup returns the names of workspaces in the group, and whether
// the group exists. A declared-but-empty group returns (nil, true).
func workspacesInGroup(cfg model.Config, name string) ([]string, bool) {
	found := false
	for _, g := range cfg.Groups {
		if g.Name == name {
			found = true
			break
		}
	}
	// Workspaces can reference a group without it being declared in [[groups]].
	var names []string
	for _, w := range cfg.Workspaces {
		if w.Group == name {
			names = append(names, w.Name)
			found = true
		}
	}
	return names, found
}

func openPicker(cfg model.Config, restrict []string) error {
	var res tui.Result
	var err error
	if restrict != nil {
		res, err = tui.RunRestrictedPicker(cfg, restrict)
	} else {
		res, err = tui.RunPicker(cfg)
	}
	if err != nil {
		return err
	}
	switch res.Action {
	case tui.ActionChoose:
		emitChosen(res.Path)
	case tui.ActionAdd:
		// Trigger add on cwd. Reload cfg from disk after.
		return runAdd(".", 0)
	case tui.ActionNone:
		// cancelled
	}
	return nil
}

func emitChosen(p string) {
	fmt.Fprintln(os.Stdout, p)
}

// ---- add ----

func cmdAdd() *cobra.Command {
	var depth int
	c := &cobra.Command{
		Use:   "add [path]",
		Short: "Save a directory as a workspace (git-aware, worktree-aware)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			return runAdd(target, depth)
		},
	}
	c.Flags().IntVar(&depth, "depth", 0, "scan depth (default: from config)")
	return c
}

func runAdd(target string, depth int) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if depth == 0 {
		depth = cfg.Scan.Depth
	}
	candidates, err := buildCandidates(cfg, abs, depth)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Fprintln(out(), "ws: nothing new to add (already saved or no projects found)")
		return nil
	}
	results, ok, err := tui.RunAddWizard(cfg, candidates)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(out(), "ws: add cancelled")
		return nil
	}
	applyAdd(&cfg, results)
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintln(out(), "ws: saved.")
	return nil
}

// buildCandidates scans for repos under abs, attaching worktrees, deduping
// against the current config per the worktree rules.
func buildCandidates(cfg model.Config, abs string, depth int) ([]tui.Candidate, error) {
	known := map[string]*model.Workspace{}
	for i := range cfg.Workspaces {
		w := &cfg.Workspaces[i]
		known[filepath.Clean(w.Path)] = w
		for j := range w.Worktrees {
			known[filepath.Clean(w.Worktrees[j].Path)] = w
		}
	}

	if git.Available() && git.IsRepo(abs) {
		root, err := git.CommonRoot(abs)
		if err != nil {
			return nil, err
		}
		root = filepath.Clean(root)
		if _, ok := known[root]; ok {
			return candidatesFromExisting(root, known), nil
		}
		return candidatesFromRepo(root)
	}

	hits, err := scan.FindRepos(abs, depth, cfg.Scan.Ignores)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		if _, ok := known[filepath.Clean(abs)]; ok {
			return nil, nil
		}
		return []tui.Candidate{{
			Path:        abs,
			DefaultName: filepath.Base(abs),
			Icon:        string(detect.DetectType(abs)),
		}}, nil
	}
	var out []tui.Candidate
	for _, h := range hits {
		root := filepath.Clean(h.Root)
		if _, ok := known[root]; ok {
			out = append(out, candidatesFromExisting(root, known)...)
			continue
		}
		cs, err := candidatesFromRepo(root)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	return out, nil
}

func candidatesFromRepo(root string) ([]tui.Candidate, error) {
	var out []tui.Candidate
	icon := string(detect.DetectType(root))
	out = append(out, tui.Candidate{
		Path:        root,
		DefaultName: filepath.Base(root),
		Icon:        icon,
	})
	if !git.Available() {
		return out, nil
	}
	entries, err := git.Worktrees(root)
	if err != nil || len(entries) <= 1 {
		return out, nil
	}
	for _, e := range entries {
		if filepath.Clean(e.Path) == root {
			continue
		}
		out = append(out, tui.Candidate{
			Path:        e.Path,
			DefaultName: filepath.Base(e.Path),
			IsWorktree:  true,
			ParentRoot:  root,
			Icon:        string(detect.DetectType(e.Path)),
		})
	}
	return out, nil
}

func candidatesFromExisting(root string, known map[string]*model.Workspace) []tui.Candidate {
	if !git.Available() {
		return nil
	}
	entries, err := git.Worktrees(root)
	if err != nil {
		return nil
	}
	var out []tui.Candidate
	for _, e := range entries {
		if filepath.Clean(e.Path) == root {
			continue
		}
		if _, ok := known[filepath.Clean(e.Path)]; ok {
			continue
		}
		out = append(out, tui.Candidate{
			Path:        e.Path,
			DefaultName: filepath.Base(e.Path),
			IsWorktree:  true,
			ParentRoot:  root,
			Icon:        string(detect.DetectType(e.Path)),
		})
	}
	return out
}

func applyAdd(cfg *model.Config, results []tui.Candidate) {
	hasGroup := map[string]bool{}
	for _, g := range cfg.Groups {
		hasGroup[g.Name] = true
	}
	byPath := map[string]*model.Workspace{}
	for i := range cfg.Workspaces {
		byPath[filepath.Clean(cfg.Workspaces[i].Path)] = &cfg.Workspaces[i]
	}

	for _, c := range results {
		if !c.Selected {
			continue
		}
		if c.Group != "" && !hasGroup[c.Group] {
			cfg.Groups = append(cfg.Groups, model.Group{Name: c.Group})
			hasGroup[c.Group] = true
		}
		if c.IsWorktree {
			parent, ok := byPath[filepath.Clean(c.ParentRoot)]
			if !ok {
				root := model.Workspace{
					Name:  filepath.Base(c.ParentRoot),
					Path:  c.ParentRoot,
					Group: c.Group,
					Icon:  string(detect.DetectType(c.ParentRoot)),
				}
				cfg.Workspaces = append(cfg.Workspaces, root)
				byPath[filepath.Clean(c.ParentRoot)] = &cfg.Workspaces[len(cfg.Workspaces)-1]
				parent = byPath[filepath.Clean(c.ParentRoot)]
			}
			// c.Name holds the user-entered suffix (may be empty). Prefix it
			// with the parent's alias so the worktree is globally resolvable
			// as `${parent}/${suffix}`. Empty = unaliased (still saved).
			finalName := ""
			if c.Name != "" {
				finalName = parent.Name + "/" + c.Name
			}
			parent.Worktrees = append(parent.Worktrees, model.Worktree{
				Name: finalName, Path: c.Path, Icon: c.Icon,
			})
			continue
		}
		if _, ok := byPath[filepath.Clean(c.Path)]; ok {
			continue
		}
		w := model.Workspace{
			Name:  c.Name,
			Path:  c.Path,
			Group: c.Group,
			Icon:  c.Icon,
		}
		cfg.Workspaces = append(cfg.Workspaces, w)
		byPath[filepath.Clean(c.Path)] = &cfg.Workspaces[len(cfg.Workspaces)-1]
	}
}

// ---- rm ----

func cmdRemove() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"remove"},
		Short:   "Remove a workspace (or worktree) by name",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			for i := range cfg.Workspaces {
				w := &cfg.Workspaces[i]
				for j, wt := range w.Worktrees {
					if wt.Name == name {
						w.Worktrees = append(w.Worktrees[:j], w.Worktrees[j+1:]...)
						fmt.Fprintln(out(), "ws: removed worktree", name)
						return config.Save(cfg)
					}
				}
			}
			for i, w := range cfg.Workspaces {
				if w.Name == name {
					cfg.Workspaces = append(cfg.Workspaces[:i], cfg.Workspaces[i+1:]...)
					fmt.Fprintln(out(), "ws: removed workspace", name)
					return config.Save(cfg)
				}
			}
			fmt.Fprintf(os.Stderr, "ws: no workspace named %q\n", name)
			os.Exit(1)
			return nil
		},
	}
}

// ---- list ----

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print all workspaces",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Workspaces) == 0 {
				fmt.Fprintln(out(), "(no workspaces — run `ws add` to create one)")
				return nil
			}
			groups := map[string][]model.Workspace{}
			for _, w := range cfg.Workspaces {
				groups[w.Group] = append(groups[w.Group], w)
			}
			keys := make([]string, 0, len(groups))
			for k := range groups {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, g := range keys {
				label := g
				if label == "" {
					label = "(ungrouped)"
				}
				fmt.Fprintln(out(), "#", label)
				for _, w := range groups[g] {
					fmt.Fprintf(out(), "  %-20s %s\n", w.Name, w.Path)
					for _, wt := range w.Worktrees {
						name := wt.Name
						if name == "" {
							name = "(" + filepath.Base(wt.Path) + ")"
						}
						fmt.Fprintf(out(), "    ↳ %-16s %s\n", name, wt.Path)
					}
				}
			}
			return nil
		},
	}
}

// ---- init ----

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init <shell>",
		Short: "Print shell integration script (bash|zsh|fish)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shell.InitScript(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, s)
			return nil
		},
	}
}

// ---- refresh ----

func cmdRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [name]",
		Short: "Re-detect icons and worktrees for all (or one) workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			changed := 0
			for i := range cfg.Workspaces {
				w := &cfg.Workspaces[i]
				if target != "" && w.Name != target {
					continue
				}
				icon := string(detect.DetectType(w.Path))
				if icon != w.Icon {
					w.Icon = icon
					changed++
				}
				if git.Available() && git.IsRepo(w.Path) {
					entries, err := git.Worktrees(w.Path)
					if err == nil {
						existing := map[string]bool{}
						for _, wt := range w.Worktrees {
							existing[filepath.Clean(wt.Path)] = true
						}
						for _, e := range entries {
							if filepath.Clean(e.Path) == filepath.Clean(w.Path) {
								continue
							}
							if existing[filepath.Clean(e.Path)] {
								continue
							}
							// refresh auto-discovers worktrees unaliased; user can
							// rename later or re-run `ws add` to alias interactively.
							w.Worktrees = append(w.Worktrees, model.Worktree{
								Name: "",
								Path: e.Path,
								Icon: string(detect.DetectType(e.Path)),
							})
							changed++
						}
					}
				}
				for j := range w.Worktrees {
					wt := &w.Worktrees[j]
					icon := string(detect.DetectType(wt.Path))
					if icon != wt.Icon {
						wt.Icon = icon
						changed++
					}
				}
			}
			if changed > 0 {
				if err := config.Save(cfg); err != nil {
					return err
				}
			}
			fmt.Fprintf(out(), "ws: refresh updated %d fields\n", changed)
			return nil
		},
	}
}

// ---- pick ----

func cmdPick() *cobra.Command {
	return &cobra.Command{
		Use:   "pick",
		Short: "Open the TUI picker explicitly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			return openPicker(cfg, nil)
		},
	}
}
