package model

import "time"

type RepoStatus struct {
	Branch    string `json:"branch"`
	Upstream  string `json:"upstream,omitempty"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Staged    int    `json:"staged"`
	Modified  int    `json:"modified"`
	Untracked int    `json:"untracked"`
	Conflicts int    `json:"conflicts"`
}

type Step struct {
	Command    string `json:"command"`
	Label      string `json:"label,omitempty"`
	Phase      string `json:"phase"`
	Result     string `json:"result"`
	ExitCode   int    `json:"exit_code"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}

type State struct {
	Sequence  int64      `json:"sequence"`
	Repo      string     `json:"repo"`
	RepoName  string     `json:"repo_name"`
	Phase     string     `json:"phase"`
	Result    string     `json:"result"`
	StartedAt int64      `json:"started_at"`
	CloseAt   int64      `json:"close_at,omitempty"`
	Status    RepoStatus `json:"status"`
	Steps     []Step     `json:"steps"`
}

func NowMillis() int64 {
	return time.Now().UnixMilli()
}
