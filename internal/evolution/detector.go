package evolution

import (
	"encoding/json"
	"log/slog"
	"time"
)

// InterventionType classifies what kind of human intervention occurred.
type InterventionType string

const (
	TypeClear           InterventionType = "clear"
	TypeNonIntervention InterventionType = "non_intervention"
)

// Intervention records a detected human intervention on harness output.
type Intervention struct {
	ID         string
	JobID      string
	RepoID     string
	Role       string
	Type       InterventionType
	Evidence   string // JSON: files changed, commit SHAs
	DetectedAt time.Time
}

// CommitInfo carries metadata about a single commit for intervention analysis.
type CommitInfo struct {
	SHA          string
	Author       string
	FilesChanged []string
	HasDiff      bool
}

// evidence is the internal structure serialised into Intervention.Evidence.
type evidence struct {
	SHAs         []string `json:"shas"`
	FilesChanged []string `json:"files_changed"`
}

// Detect classifies post-job activity as intervention or non-intervention.
// It examines commits after the harness's last commit to find human changes.
// Returns the intervention type and a JSON evidence string.
func Detect(harnessAuthor string, commits []CommitInfo) (InterventionType, string) {
	var humanSHAs []string
	var humanFiles []string
	seen := make(map[string]struct{})

	for _, c := range commits {
		if c.Author == harnessAuthor {
			continue
		}
		if !c.HasDiff || len(c.FilesChanged) == 0 {
			continue
		}
		humanSHAs = append(humanSHAs, c.SHA)
		for _, f := range c.FilesChanged {
			if _, ok := seen[f]; !ok {
				seen[f] = struct{}{}
				humanFiles = append(humanFiles, f)
			}
		}
	}

	if len(humanSHAs) == 0 {
		ev := marshalEvidence(evidence{})
		slog.Debug("evolution: no human code changes detected", "harness_author", harnessAuthor, "commits", len(commits))
		return TypeNonIntervention, ev
	}

	ev := marshalEvidence(evidence{SHAs: humanSHAs, FilesChanged: humanFiles})
	slog.Debug("evolution: human intervention detected",
		"harness_author", harnessAuthor,
		"human_commits", len(humanSHAs),
		"files_changed", len(humanFiles),
	)
	return TypeClear, ev
}

func marshalEvidence(e evidence) string {
	b, err := json.Marshal(e)
	if err != nil {
		return "{}"
	}
	return string(b)
}
