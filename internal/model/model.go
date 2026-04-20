package model

type Config struct {
	Schema     int         `toml:"schema"`
	Scan       ScanConfig  `toml:"scan"`
	Groups     []Group     `toml:"groups"`
	Workspaces []Workspace `toml:"workspaces"`
}

type ScanConfig struct {
	Depth   int      `toml:"depth"`
	Ignores []string `toml:"ignores"`
}

type Group struct {
	Name string `toml:"name"`
}

type Workspace struct {
	Name      string     `toml:"name"`
	Path      string     `toml:"path"`
	Group     string     `toml:"group,omitempty"`
	Icon      string     `toml:"icon,omitempty"`
	Worktrees []Worktree `toml:"worktrees,omitempty"`
}

type Worktree struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
	Icon string `toml:"icon,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Schema: 1,
		Scan: ScanConfig{
			Depth:   2,
			Ignores: []string{"node_modules", ".venv", "venv", "target", "dist", "build", ".git", ".next", ".cache"},
		},
	}
}
