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

func NowMillis() int64 {
	return time.Now().UnixMilli()
}
