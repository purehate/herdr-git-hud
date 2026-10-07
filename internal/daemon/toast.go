package daemon

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/purehate/herdr-git-hud/internal/model"
)

const maxHeadWidth = 50

// writesHistory lists commands whose resulting commit is worth naming.
var writesHistory = map[string]bool{
	"am": true, "cherry-pick": true, "commit": true, "merge": true,
	"pull": true, "rebase": true, "revert": true,
}

// toastText renders the title and one-line body for a finished operation.
func toastText(operation *pendingOperation, code int, status model.RepoStatus, head string, elapsed time.Duration) (string, string) {
	label := operation.Label
	if label == "" {
		label = "git " + operation.Command
	}
	repo := filepath.Base(operation.Repo)
	title := "✓ " + label + " · " + repo
	var parts []string
	if code != 0 {
		title = "✗ " + label + " failed · " + repo
		parts = append(parts, fmt.Sprintf("exit %d", code))
		if operation.Phase != "" {
			parts = append(parts, operation.Phase)
		}
	} else if head != "" {
		parts = append(parts, clip(head, maxHeadWidth))
	}
	parts = append(parts, branchParts(status)...)
	parts = append(parts, formatElapsed(elapsed))
	return title, strings.Join(parts, " · ")
}

func branchParts(status model.RepoStatus) []string {
	if status.Branch == "" {
		return nil
	}
	branch := status.Branch
	if branch == "(detached)" {
		branch = "detached HEAD"
	}
	if status.Ahead > 0 {
		branch += fmt.Sprintf(" ↑%d", status.Ahead)
	}
	if status.Behind > 0 {
		branch += fmt.Sprintf(" ↓%d", status.Behind)
	}
	parts := []string{branch}
	if status.Conflicts > 0 {
		parts = append(parts, fmt.Sprintf("%d conflicts", status.Conflicts))
	}
	return parts
}

func formatElapsed(elapsed time.Duration) string {
	elapsed = max(0, elapsed)
	if elapsed < time.Minute {
		return fmt.Sprintf("%.1fs", elapsed.Seconds())
	}
	return elapsed.Truncate(time.Second).String()
}

func clip(value string, width int) string {
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	return string([]rune(value)[:width-1]) + "…"
}
