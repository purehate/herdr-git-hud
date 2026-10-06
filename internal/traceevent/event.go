package traceevent

import "strings"

type Event struct {
	HUDType        string   `json:"hud_type,omitempty"`
	Operation      string   `json:"operation,omitempty"`
	OperationLabel string   `json:"operation_label,omitempty"`
	Repo           string   `json:"repo,omitempty"`
	ExitCode       int      `json:"exit_code,omitempty"`
	Event          string   `json:"event"`
	SID            string   `json:"sid"`
	Argv           []string `json:"argv"`
	Worktree       string   `json:"worktree"`
	Name           string   `json:"name"`
	Hierarchy      string   `json:"hierarchy"`
	Category       string   `json:"category"`
	Label          string   `json:"label"`
	Code           int      `json:"code"`
}

var visibleCommands = map[string]bool{
	"add": true, "am": true, "apply": true, "branch": true,
	"checkout": true, "cherry-pick": true, "clean": true,
	"clone": true, "commit": true, "fetch": true, "merge": true,
	"pull": true, "push": true, "rebase": true, "reset": true,
	"restore": true, "revert": true, "stash": true, "switch": true,
	"tag": true,
}

func (e Event) RootSID() string {
	root, _, _ := strings.Cut(e.SID, "/")
	return root
}

func (e Event) IsRoot() bool {
	return e.SID != "" && !strings.Contains(e.SID, "/")
}

func IsVisibleOperation(name string, argv []string) bool {
	if !visibleCommands[name] {
		return false
	}
	switch name {
	case "branch":
		return !hasAnyArgument(argv, "--list", "-l", "--show-current", "-a", "--all", "-r", "--remotes")
	case "stash":
		return !hasAnyArgument(argv, "list", "show")
	case "tag":
		return !hasAnyArgument(argv, "--list", "-l")
	default:
		return true
	}
}

func hasAnyArgument(argv []string, candidates ...string) bool {
	for _, argument := range argv {
		for _, candidate := range candidates {
			if argument == candidate {
				return true
			}
		}
	}
	return false
}

func Phase(e Event) string {
	text := strings.ToLower(e.Category + " " + e.Label + " " + strings.Join(e.Argv, " "))
	for needle, phase := range phaseMatchers {
		if strings.Contains(text, needle) {
			return phase
		}
	}
	return ""
}

var phaseMatchers = map[string]string{
	"pre-commit":      "Running pre-commit checks",
	"commit-msg":      "Validating commit message",
	"pre-push":        "Running pre-push checks",
	"pack-objects":    "Packing objects",
	"send-pack":       "Sending objects",
	"receive-pack":    "Updating remote",
	"fetch-pack":      "Fetching objects",
	"index-pack":      "Indexing received objects",
	"unpack-objects":  "Unpacking objects",
	"checkout":        "Updating working tree",
	"merge":           "Merging changes",
	"rebase":          "Replaying commits",
	"transport_push":  "Contacting remote",
	"transport_fetch": "Contacting remote",
}
