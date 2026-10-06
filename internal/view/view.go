package view

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/purehate/herdr-git-hud/internal/model"
)

const (
	reset  = "\x1b[0m"
	green  = "\x1b[38;2;20;226;26m"
	yellow = "\x1b[38;2;232;253;46m"
	red    = "\x1b[38;2;243;139;168m"
	muted  = "\x1b[38;2;123;132;150m"
	white  = "\x1b[38;2;237;241;243m"
)

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func Run() error {
	stateFile, err := resolveStateFile()
	if err != nil {
		return err
	}
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h", reset)

	frame := 0
	for {
		state, err := readState(stateFile)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		render(state, frame)
		frame++
		if state.CloseAt > 0 && model.NowMillis() >= state.CloseAt {
			return nil
		}
		time.Sleep(90 * time.Millisecond)
	}
}

func resolveStateFile() (string, error) {
	if value := os.Getenv("GIT_HUD_STATE_FILE"); value != "" {
		return value, nil
	}
	paths, err := model.PathsFromEnv()
	if err != nil {
		return "", err
	}
	return paths.StateFile, nil
}

func readState(path string) (model.State, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return model.State{}, err
	}
	defer func() { _ = file.Close() }()
	var state model.State
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return model.State{}, fmt.Errorf("decode HUD state: %w", err)
	}
	return state, nil
}

func render(state model.State, frame int) {
	width := 72
	inner := width - 4
	lines := make([]string, 0, 14)
	lines = append(lines, titleLine(state, frame))
	lines = append(lines, "")
	lines = append(lines, clip(state.Repo, inner))
	lines = append(lines, repoStatus(state.Status))
	lines = append(lines, "")
	lines = append(lines, stepLines(state, frame)...)
	lines = append(lines, "")
	lines = append(lines, elapsedLine(state))

	var output strings.Builder
	output.WriteString("\x1b[2J\x1b[H\n\n")
	output.WriteString(green + "  ╭" + strings.Repeat("─", width-2) + "╮" + reset + "\n")
	for _, line := range lines {
		output.WriteString(green + "  │ " + reset)
		output.WriteString(padVisible(line, inner))
		output.WriteString(green + " │" + reset + "\n")
	}
	output.WriteString(green + "  ╰" + strings.Repeat("─", width-2) + "╯" + reset + "\n")
	fmt.Print(output.String())
}

func titleLine(state model.State, frame int) string {
	icon := spinner[frame%len(spinner)]
	color := yellow
	label := strings.ToUpper(state.Phase)
	switch state.Result {
	case "success":
		icon, color, label = "✓", green, "GIT OPERATION COMPLETE"
	case "failure":
		icon, color, label = "✗", red, "GIT OPERATION FAILED"
	}
	return color + icon + "  " + label + reset
}

func stepLines(state model.State, frame int) []string {
	start := 0
	if len(state.Steps) > 5 {
		start = len(state.Steps) - 5
	}
	lines := make([]string, 0, len(state.Steps)-start)
	for index, step := range state.Steps[start:] {
		icon, color := "○", muted
		switch step.Result {
		case "running":
			icon, color = spinner[(frame+index)%len(spinner)], yellow
		case "success":
			icon, color = "✓", green
		case "failure":
			icon, color = "✗", red
		}
		label := "git " + step.Command
		if step.Phase != "" && step.Phase != "Completed" && step.Phase != "Failed" {
			label += " · " + step.Phase
		}
		lines = append(lines, color+icon+reset+"  "+white+label+reset)
	}
	return lines
}

func repoStatus(status model.RepoStatus) string {
	parts := []string{green + branchName(status.Branch) + reset}
	if status.Upstream != "" {
		parts = append(parts, muted+"→ "+status.Upstream+reset)
	}
	if status.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%s↑%d%s", yellow, status.Ahead, reset))
	}
	if status.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%s↓%d%s", yellow, status.Behind, reset))
	}
	changes := status.Staged + status.Modified + status.Untracked + status.Conflicts
	if changes > 0 {
		parts = append(parts, fmt.Sprintf("%s+%d ~%d ?%d !%d%s", muted,
			status.Staged, status.Modified, status.Untracked, status.Conflicts, reset))
	}
	return strings.Join(parts, "  ")
}

func elapsedLine(state model.State) string {
	end := model.NowMillis()
	if state.CloseAt > 0 {
		end = state.CloseAt - 2200
	}
	duration := time.Duration(max(0, end-state.StartedAt)) * time.Millisecond
	return muted + fmt.Sprintf("elapsed %.1fs", duration.Seconds()) + reset
}

func branchName(branch string) string {
	if branch == "" || branch == "(detached)" {
		return "detached HEAD"
	}
	return branch
}

func clip(value string, width int) string {
	if utf8.RuneCountInString(value) <= width {
		return muted + value + reset
	}
	runes := []rune(value)
	return muted + "…" + string(runes[len(runes)-width+1:]) + reset
}

func padVisible(value string, width int) string {
	plain := stripANSI(value)
	visibleWidth := utf8.RuneCountInString(plain)
	if visibleWidth >= width {
		return value
	}
	return value + strings.Repeat(" ", width-visibleWidth)
}

func stripANSI(value string) string {
	for _, code := range []string{reset, green, yellow, red, muted, white} {
		value = strings.ReplaceAll(value, code, "")
	}
	return value
}
