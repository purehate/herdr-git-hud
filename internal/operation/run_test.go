package operation

import (
	"bytes"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		label string
		want  []string
	}{
		{
			name:  "explicit label",
			args:  []string{"--label", "Odoo tests", "--", "odoo-test", "trustedsec"},
			label: "Odoo tests",
			want:  []string{"odoo-test", "trustedsec"},
		},
		{
			name:  "command becomes label",
			args:  []string{"--", "gh", "pr", "checks"},
			label: "gh pr checks",
			want:  []string{"gh", "pr", "checks"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parse(test.args)
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if parsed.Label != test.label || strings.Join(parsed.Args, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("parse() = %#v, want label %q args %#v", parsed, test.label, test.want)
			}
		})
	}
}

func TestRunPreservesOutputAndExitCode(t *testing.T) {
	t.Setenv("HERDR_PLUGIN_STATE_DIR", t.TempDir())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode, err := Run(
		[]string{"--label", "failing test", "--", "/bin/sh", "-c", "printf output; printf error >&2; exit 7"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if exitCode != 7 {
		t.Fatalf("Run() exit code = %d, want 7", exitCode)
	}
	if stdout.String() != "output" || stderr.String() != "error" {
		t.Fatalf("Run() output = (%q, %q)", stdout.String(), stderr.String())
	}
}

func TestParseRejectsMissingCommand(t *testing.T) {
	if _, err := parse([]string{"--label", "nothing", "--"}); err == nil {
		t.Fatal("parse() accepted a missing command")
	}
}
