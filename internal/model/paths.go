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
		var err error
		stateDir, err = defaultStateDir()
		if err != nil {
			return Paths{}, err
		}
	}
	return Paths{
		StateDir:  stateDir,
		Socket:    filepath.Join(stateDir, "events.sock"),
		StateFile: filepath.Join(stateDir, "hud.json"),
		PIDFile:   filepath.Join(stateDir, "daemon.pid"),
		LogFile:   filepath.Join(stateDir, "daemon.log"),
	}, nil
}

func defaultStateDir() (string, error) {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "herdr", "plugins", "purehate.git-hud"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "herdr", "plugins", "purehate.git-hud"), nil
}
