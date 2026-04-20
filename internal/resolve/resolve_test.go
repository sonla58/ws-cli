package resolve

import (
	"testing"

	"github.com/longnguyen/ws-cli/internal/model"
)

func testCfg() model.Config {
	return model.Config{
		Workspaces: []model.Workspace{
			{Name: "api", Path: "/a", Worktrees: []model.Worktree{{Name: "api-feat", Path: "/a-feat"}}},
			{Name: "web", Path: "/w"},
			{Name: "worker", Path: "/wk"},
		},
	}
}

func TestResolveExact(t *testing.T) {
	h := Resolve(testCfg(), "api")
	if len(h) != 1 || h[0].Path() != "/a" {
		t.Fatalf("exact: %+v", h)
	}
}

func TestResolveWorktreeByName(t *testing.T) {
	h := Resolve(testCfg(), "api-feat")
	if len(h) != 1 || h[0].Path() != "/a-feat" {
		t.Fatalf("wt exact: %+v", h)
	}
}

func TestResolvePrefixMultiple(t *testing.T) {
	h := Resolve(testCfg(), "w")
	// web + worker match prefix. api-feat does not.
	if len(h) != 2 {
		t.Fatalf("prefix count: %+v", h)
	}
}

func TestResolveFuzzy(t *testing.T) {
	h := Resolve(testCfg(), "wrk")
	if len(h) == 0 || h[0].Name() != "worker" {
		t.Fatalf("fuzzy: %+v", h)
	}
}

func TestResolveEmpty(t *testing.T) {
	if h := Resolve(testCfg(), ""); h != nil {
		t.Fatalf("empty query should return nil, got %+v", h)
	}
}
