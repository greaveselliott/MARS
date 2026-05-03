package tickets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	StatusBacklog    = "backlog"
	StatusInProgress = "in-progress"
	StatusDone       = "done"

	DefaultStaleInProgressAfter = 7 * 24 * time.Hour
)

var ticketIDRe = regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

// Ticket is the repo-visible markdown ticket state used by gates, scanner, and doctor.
type Ticket struct {
	ID            string
	Name          string
	RelPath       string
	Status        string
	Title         string
	Priority      string
	Kind          string
	WorkType      string
	Owner         string
	LastAttempt   string
	LastAttemptAt time.Time
	Blocker       string
	BlockedBy     []string
	TraceID       string
	NextAction    string
	ModTime       time.Time
}

// List reads docs/tickets/{backlog,in-progress,done} from repoRoot.
func List(repoRoot string) ([]Ticket, error) {
	var out []Ticket
	for _, status := range []string{StatusBacklog, StatusInProgress, StatusDone} {
		dir := filepath.Join(repoRoot, "docs", "tickets", status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("tickets: read %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "README.md" {
				continue
			}
			t, err := readTicket(filepath.Join(dir, entry.Name()), repoRoot, status, entry.Name())
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		}
	}
	Sort(out)
	return out, nil
}

// ListStatus reads tickets from one ticket status directory.
func ListStatus(repoRoot, status string) ([]Ticket, error) {
	all, err := List(repoRoot)
	if err != nil {
		return nil, err
	}
	var out []Ticket
	for _, t := range all {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out, nil
}

// EligibleInProgress returns in-progress tickets that are not explicitly blocked.
func EligibleInProgress(repoRoot string) ([]Ticket, error) {
	all, err := ListStatus(repoRoot, StatusInProgress)
	if err != nil {
		return nil, err
	}
	var out []Ticket
	for _, t := range all {
		if t.EligibleInProgress() {
			out = append(out, t)
		}
	}
	return out, nil
}

// StaleInProgress returns eligible in-progress tickets older than threshold.
func StaleInProgress(repoRoot string, now time.Time, threshold time.Duration) ([]Ticket, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if threshold <= 0 {
		threshold = DefaultStaleInProgressAfter
	}
	inProgress, err := ListStatus(repoRoot, StatusInProgress)
	if err != nil {
		return nil, err
	}
	var out []Ticket
	for _, t := range inProgress {
		if !t.EligibleInProgress() {
			continue
		}
		last := t.LastActivity()
		if last.IsZero() || last.After(now) {
			continue
		}
		if now.Sub(last) >= threshold {
			out = append(out, t)
		}
	}
	return out, nil
}

// Sort orders tickets by ticket id where possible, then by filename.
func Sort(ts []Ticket) {
	sort.Slice(ts, func(i, j int) bool {
		leftID := TicketNumber(ts[i].ID)
		rightID := TicketNumber(ts[j].ID)
		if leftID != rightID {
			if leftID == 0 {
				return false
			}
			if rightID == 0 {
				return true
			}
			return leftID < rightID
		}
		return ts[i].Name < ts[j].Name
	})
}

// TicketNumber extracts the numeric suffix from a ticket id.
func TicketNumber(id string) int {
	idx := strings.LastIndex(id, "-")
	if idx < 0 {
		return 0
	}
	var n int
	for _, r := range id[idx+1:] {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func (t Ticket) Blocked() bool {
	return meaningful(t.Blocker) || len(t.BlockedBy) > 0
}

func (t Ticket) EligibleInProgress() bool {
	return t.Status == StatusInProgress && !t.Blocked()
}

func (t Ticket) LastActivity() time.Time {
	if !t.LastAttemptAt.IsZero() {
		return t.LastAttemptAt
	}
	return t.ModTime
}

func (t Ticket) LastActivityLabel() string {
	if !t.LastAttemptAt.IsZero() {
		return t.LastAttemptAt.Format(time.RFC3339)
	}
	if !t.ModTime.IsZero() {
		return t.ModTime.Format(time.RFC3339)
	}
	return "unknown"
}

func readTicket(path, repoRoot, status, name string) (Ticket, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Ticket{}, fmt.Errorf("tickets: stat %s: %w", path, err)
	}
	frontmatter := readFrontmatter(path)
	rel, _ := filepath.Rel(repoRoot, path)
	id := strings.TrimSpace(frontmatter["id"])
	if id == "" {
		id = IDFromName(name)
	}
	lastAttempt := strings.Trim(strings.TrimSpace(frontmatter["last_attempt"]), `"'`)
	return Ticket{
		ID:            id,
		Name:          name,
		RelPath:       filepath.ToSlash(rel),
		Status:        status,
		Title:         strings.Trim(strings.TrimSpace(frontmatter["title"]), `"'`),
		Priority:      strings.Trim(strings.TrimSpace(frontmatter["priority"]), `"'`),
		Kind:          strings.Trim(strings.TrimSpace(frontmatter["kind"]), `"'`),
		WorkType:      strings.Trim(strings.TrimSpace(frontmatter["work_type"]), `"'`),
		Owner:         strings.Trim(strings.TrimSpace(frontmatter["owner"]), `"'`),
		LastAttempt:   lastAttempt,
		LastAttemptAt: parseTicketTime(lastAttempt),
		Blocker:       strings.Trim(strings.TrimSpace(frontmatter["blocker"]), `"'`),
		BlockedBy:     parseInlineList(frontmatter["blocked_by"]),
		TraceID:       strings.Trim(strings.TrimSpace(frontmatter["trace_id"]), `"'`),
		NextAction:    strings.Trim(strings.TrimSpace(frontmatter["next_action"]), `"'`),
		ModTime:       info.ModTime().UTC(),
	}, nil
}

func readFrontmatter(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if !inFrontmatter || strings.HasPrefix(line, " ") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func IDFromName(name string) string {
	if match := ticketIDRe.FindStringSubmatch(name); len(match) == 2 {
		return match[1]
	}
	return ""
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	if !meaningful(value) {
		return nil
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if meaningful(part) {
			out = append(out, part)
		}
	}
	return out
}

func parseTicketTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if !meaningful(value) {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func meaningful(value string) bool {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	switch strings.ToLower(value) {
	case "", "[]", "none", "null", "nil", "tbd", "todo":
		return false
	default:
		return true
	}
}
