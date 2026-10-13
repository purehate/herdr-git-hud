package traceevent

import (
	"encoding/json"
	"testing"
)

func TestRootSID(t *testing.T) {
	event := Event{SID: "parent/child/grandchild"}
	if got := event.RootSID(); got != "parent" {
		t.Fatalf("RootSID() = %q, want parent", got)
	}
	if event.IsRoot() {
		t.Fatal("nested SID reported as root")
	}
}

func TestVisibleCommands(t *testing.T) {
	if !IsVisibleOperation("push", []string{"git", "push"}) {
		t.Fatal("push must be visible")
	}
	if IsVisibleOperation("status", []string{"git", "status"}) {
		t.Fatal("status must be filtered")
	}
	if IsVisibleOperation("branch", []string{"git", "branch", "--show-current"}) {
		t.Fatal("read-only branch query must be filtered")
	}
	if !IsVisibleOperation("branch", []string{"git", "branch", "release"}) {
		t.Fatal("branch creation must be visible")
	}
}

func TestNoisyOperationsStayQuiet(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{"add", []string{"git", "add", "."}, false},
		{"fetch", []string{"git", "fetch", "--all"}, false},
		{"branch", []string{"git", "branch"}, false},
		{"branch", []string{"git", "branch", "-vv"}, false},
		{"branch", []string{"git", "branch", "--merged", "main"}, false},
		{"branch", []string{"git", "-C", "/src/app", "branch", "-d", "old"}, true},
		{"tag", []string{"git", "tag"}, false},
		{"tag", []string{"git", "tag", "--points-at", "HEAD"}, false},
		{"tag", []string{"git", "tag", "-a", "v1.0.0", "-m", "release"}, true},
	}

	for _, test := range tests {
		if got := IsVisibleOperation(test.name, test.argv); got != test.want {
			t.Errorf("IsVisibleOperation(%q, %q) = %t, want %t", test.name, test.argv, got, test.want)
		}
	}
}

func TestPhase(t *testing.T) {
	event := Event{Event: "child_start", Argv: []string{"git", "pack-objects"}}
	if got := Phase(event); got != "Packing objects" {
		t.Fatalf("Phase() = %q, want Packing objects", got)
	}
}

func TestRepoFieldAcceptsNumberAndString(t *testing.T) {
	// Trace2 def_repo carries a numeric repo index with the path in worktree.
	var trace Event
	if err := json.Unmarshal([]byte(`{"event":"def_repo","repo":1,"worktree":"/src/app"}`), &trace); err != nil {
		t.Fatalf("def_repo unmarshal: %v", err)
	}
	if trace.Event != "def_repo" || trace.Worktree != "/src/app" {
		t.Fatalf("def_repo parsed as event=%q worktree=%q", trace.Event, trace.Worktree)
	}

	// HUD wrapper events carry the repository path as a string.
	var hud Event
	if err := json.Unmarshal([]byte(`{"hud_type":"operation_start","repo":"/src/app"}`), &hud); err != nil {
		t.Fatalf("hud unmarshal: %v", err)
	}
	if hud.Repo != "/src/app" {
		t.Fatalf("hud repo = %q, want /src/app", hud.Repo)
	}
}
