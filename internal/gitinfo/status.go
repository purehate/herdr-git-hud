package gitinfo

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/purehate/herdr-git-hud/internal/model"
)

func Root(cwd string) (string, error) {
	output, err := run(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(output)), nil
}

func Collect(repo string) (model.RepoStatus, error) {
	output, err := run(repo, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return model.RepoStatus{}, err
	}
	return ParseStatus(output), nil
}

func ParseStatus(output string) model.RepoStatus {
	status := model.RepoStatus{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		parseLine(scanner.Text(), &status)
	}
	return status
}

func parseLine(line string, status *model.RepoStatus) {
	switch {
	case strings.HasPrefix(line, "# branch.head "):
		status.Branch = strings.TrimPrefix(line, "# branch.head ")
	case strings.HasPrefix(line, "# branch.upstream "):
		status.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
	case strings.HasPrefix(line, "# branch.ab "):
		parseAheadBehind(strings.TrimPrefix(line, "# branch.ab "), status)
	case strings.HasPrefix(line, "? "):
		status.Untracked++
	case strings.HasPrefix(line, "u "):
		status.Conflicts++
	case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
		parseChange(line, status)
	}
}

func parseAheadBehind(value string, status *model.RepoStatus) {
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return
	}
	status.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
	status.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
}

func parseChange(line string, status *model.RepoStatus) {
	parts := strings.Fields(line)
	if len(parts) < 2 || len(parts[1]) != 2 {
		return
	}
	if parts[1][0] != '.' {
		status.Staged++
	}
	if parts[1][1] != '.' {
		status.Modified++
	}
}

func run(cwd string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"--no-optional-locks", "-C", cwd}, args...)...)
	command.Env = withoutTrace(os.Environ())
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func withoutTrace(environment []string) []string {
	filtered := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, "GIT_TRACE2_EVENT=") {
			filtered = append(filtered, item)
		}
	}
	return append(filtered, "GIT_TRACE2_EVENT=0")
}
