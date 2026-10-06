package main

import (
	"fmt"
	"os"

	"github.com/purehate/herdr-git-hud/internal/control"
	"github.com/purehate/herdr-git-hud/internal/daemon"
	"github.com/purehate/herdr-git-hud/internal/view"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "git-hud:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: herdr-git-hud <enable|disable|status|daemon|view>")
	}

	switch args[0] {
	case "enable":
		return control.Enable()
	case "disable":
		return control.Disable()
	case "status":
		return control.Status(os.Stdout)
	case "daemon":
		return daemon.Command(args[1:])
	case "view":
		return view.Run()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
