package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/longnguyen/ws-cli/internal/model"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WS_CONFIG", filepath.Join(dir, "config.toml"))

	cfg := model.DefaultConfig()
	cfg.Groups = []model.Group{{Name: "work"}}
	cfg.Workspaces = []model.Workspace{
		{Name: "api", Path: "/tmp/api", Group: "work", Icon: "node",
			Worktrees: []model.Worktree{{Name: "api-feat", Path: "/tmp/api-feat", Icon: "node"}}},
		{Name: "scripts", Path: "/tmp/scripts"},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, _, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Workspaces) != 2 || got.Workspaces[0].Name != "api" || got.Workspaces[0].Worktrees[0].Name != "api-feat" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Groups[0].Name != "work" {
		t.Fatalf("groups mismatch: %+v", got.Groups)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WS_CONFIG", filepath.Join(dir, "nope.toml"))
	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Schema != 1 || cfg.Scan.Depth != 2 {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestPathRespectsEnv(t *testing.T) {
	t.Setenv("WS_CONFIG", "/custom/path.toml")
	p, err := Path()
	if err != nil || p != "/custom/path.toml" {
		t.Fatalf("WS_CONFIG ignored: %s err=%v", p, err)
	}
	t.Setenv("WS_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	p, _ = Path()
	if p != filepath.Join("/xdg", "ws", "config.toml") {
		t.Fatalf("XDG not honored: %s", p)
	}
	_ = os.Setenv
}
