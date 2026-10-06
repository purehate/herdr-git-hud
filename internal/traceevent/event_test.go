package traceevent

import "testing"

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

func TestPhase(t *testing.T) {
	event := Event{Event: "child_start", Argv: []string{"git", "pack-objects"}}
	if got := Phase(event); got != "Packing objects" {
		t.Fatalf("Phase() = %q, want Packing objects", got)
	}
}
