package model

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	StateDir  string
	Socket    string
	StateFile string
	PIDFile   string
	LogFile   string
}

func PathsFromEnv() (Paths, error) {
	stateDir := os.Getenv("HERDR_PLUGIN_STATE_DIR")
	if stateDir == "" {
		return Paths{}, fmt.Errorf("HERDR_PLUGIN_STATE_DIR is not set")
	}
	return Paths{
		StateDir:  stateDir,
		Socket:    filepath.Join(stateDir, "events.sock"),
		StateFile: filepath.Join(stateDir, "hud.json"),
		PIDFile:   filepath.Join(stateDir, "daemon.pid"),
		LogFile:   filepath.Join(stateDir, "daemon.log"),
	}, nil
}
