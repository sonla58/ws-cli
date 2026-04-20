package scan

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func mkRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFindReposDepthAndIgnore(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "a"))
	mkRepo(t, filepath.Join(root, "nested", "b"))
	mkRepo(t, filepath.Join(root, "nested", "deep", "c"))
	mkRepo(t, filepath.Join(root, "node_modules", "skipme"))

	// Depth 2: should find a, nested/b; NOT deep/c; NOT node_modules/*.
	hits, err := FindRepos(root, 2, []string{"node_modules"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, h := range hits {
		rel, _ := filepath.Rel(root, h.Root)
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	want := []string{"a", filepath.Join("nested", "b")}
	if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] {
		t.Fatalf("got %v want %v", paths, want)
	}
}

func TestFindReposRootIsRepo(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, root)
	// Even if subrepos exist, only root is returned (we stop at .git).
	mkRepo(t, filepath.Join(root, "sub"))
	hits, err := FindRepos(root, 3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Root != root {
		t.Fatalf("expected only root, got %+v", hits)
	}
}
