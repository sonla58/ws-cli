package git

import "testing"

func TestParseWorktreeList(t *testing.T) {
	in := `worktree /home/x/proj
HEAD abc123
branch refs/heads/main

worktree /home/x/proj-feat
HEAD def456
branch refs/heads/feat

worktree /home/x/proj-det
HEAD 999000
detached
`
	got := parseWorktreeList(in)
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d (%+v)", len(got), got)
	}
	if got[0].Path != "/home/x/proj" || got[0].Branch != "refs/heads/main" {
		t.Fatalf("main wrong: %+v", got[0])
	}
	if got[1].Branch != "refs/heads/feat" {
		t.Fatalf("feat wrong: %+v", got[1])
	}
	if got[2].Path != "/home/x/proj-det" || got[2].Branch != "" {
		t.Fatalf("detached wrong: %+v", got[2])
	}
}
