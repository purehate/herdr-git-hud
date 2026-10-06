package gitinfo

import "testing"

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
