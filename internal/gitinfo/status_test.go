package gitinfo

import (
	"os"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	input := `# branch.oid abc123
# branch.head main
# branch.upstream origin/main
# branch.ab +2 -1
1 M. N... 100644 100644 100644 abc abc staged.txt
1 .M N... 100644 100644 100644 abc abc modified.txt
? untracked.txt
u UU N... 100644 100644 100644 100644 abc abc abc conflict.txt
`
	status := ParseStatus(input)
	if status.Branch != "main" || status.Upstream != "origin/main" {
		t.Fatalf("unexpected branches: %#v", status)
	}
	if status.Ahead != 2 || status.Behind != 1 {
		t.Fatalf("unexpected divergence: %#v", status)
	}
	if status.Staged != 1 || status.Modified != 1 || status.Untracked != 1 || status.Conflicts != 1 {
		t.Fatalf("unexpected counts: %#v", status)
	}
}

func TestHead(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"-c", "user.name=HUD", "-c", "user.email=hud@example.invalid", "commit", "--quiet", "--allow-empty", "-m", "feat: first toast"},
	} {
		if _, err := run(repo, args...); err != nil {
			t.Fatal(err)
		}
	}

	head, err := Head(repo)
	if err != nil {
		t.Fatalf("Head() error = %v", err)
	}
	hash, subject, found := strings.Cut(head, " ")
	if !found || len(hash) < 7 || subject != "feat: first toast" {
		t.Fatalf("Head() = %q, want short hash and subject", head)
	}
}
