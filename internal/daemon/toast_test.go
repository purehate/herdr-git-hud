package daemon

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/purehate/herdr-git-hud/internal/model"
)

func TestToastText(t *testing.T) {
	tests := []struct {
		name      string
		operation pendingOperation
		code      int
		status    model.RepoStatus
		head      string
		elapsed   time.Duration
		title     string
		body      string
	}{
		{
			name:      "commit shows the new commit",
			operation: pendingOperation{Command: "commit", Repo: "/src/herdr-git-hud"},
			status:    model.RepoStatus{Branch: "main", Upstream: "origin/main", Ahead: 1},
			head:      "a1b2c3d feat: add toasts",
			elapsed:   400 * time.Millisecond,
			title:     "✓ git commit · herdr-git-hud",
			body:      "a1b2c3d feat: add toasts · main ↑1 · 0.4s",
		},
		{
			name:      "failure shows exit code and last phase",
			operation: pendingOperation{Command: "push", Repo: "/src/herdr-git-hud", Phase: "Contacting remote"},
			code:      128,
			status:    model.RepoStatus{Branch: "main", Ahead: 1, Behind: 2},
			head:      "a1b2c3d ignored on failure",
			elapsed:   2 * time.Second,
			title:     "✗ git push failed · herdr-git-hud",
			body:      "exit 128 · Contacting remote · main ↑1 ↓2 · 2.0s",
		},
		{
			name:      "wrapped operation uses its label and minutes",
			operation: pendingOperation{Label: "odoo-test trustedsec_sow", Repo: "/src/trustedsec"},
			status:    model.RepoStatus{Branch: "fix/sow"},
			elapsed:   136 * time.Second,
			title:     "✓ odoo-test trustedsec_sow · trustedsec",
			body:      "fix/sow · 2m16s",
		},
		{
			name:      "conflicts on a detached head",
			operation: pendingOperation{Command: "rebase", Repo: "/src/app"},
			code:      1,
			status:    model.RepoStatus{Branch: "(detached)", Conflicts: 2},
			elapsed:   100 * time.Millisecond,
			title:     "✗ git rebase failed · app",
			body:      "exit 1 · detached HEAD · 2 conflicts · 0.1s",
		},
		{
			name:      "outside a repository",
			operation: pendingOperation{Label: "make", Repo: "/tmp/scratch"},
			elapsed:   1500 * time.Millisecond,
			title:     "✓ make · scratch",
			body:      "1.5s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			title, body := toastText(&test.operation, test.code, test.status, test.head, test.elapsed)
			if title != test.title || body != test.body {
				t.Fatalf("toastText() = (%q, %q), want (%q, %q)", title, body, test.title, test.body)
			}
		})
	}
}

func TestToastTextClipsLongSubjects(t *testing.T) {
	head := "a1b2c3d " + strings.Repeat("x", 80)
	_, body := toastText(&pendingOperation{Command: "commit", Repo: "/src/app"}, 0, model.RepoStatus{}, head, 0)
	clipped, _, _ := strings.Cut(body, " · ")
	if utf8.RuneCountInString(clipped) != maxHeadWidth || !strings.HasSuffix(clipped, "…") {
		t.Fatalf("clipped head = %q, want %d runes ending in an ellipsis", clipped, maxHeadWidth)
	}
}
