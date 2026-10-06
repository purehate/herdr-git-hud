package main

import (
	"fmt"
	"os"

	"github.com/purehate/herdr-git-hud/internal/control"
	"github.com/purehate/herdr-git-hud/internal/daemon"
	"github.com/purehate/herdr-git-hud/internal/operation"
	"github.com/purehate/herdr-git-hud/internal/view"
)

func main() {
	exitCode, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "git-hud:", err)
	}
	os.Exit(exitCode)
}

func run(args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("usage: herdr-git-hud <enable|disable|status|daemon|run|view>")
	}

	var err error
	switch args[0] {
	case "enable":
		err = control.Enable()
	case "disable":
		err = control.Disable()
	case "status":
		err = control.Status(os.Stdout)
	case "daemon":
		err = daemon.Command(args[1:])
	case "run":
		return operation.Run(args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "view":
		err = view.Run()
	default:
		return 1, fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}
