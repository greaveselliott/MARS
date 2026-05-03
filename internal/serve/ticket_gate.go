package serve

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

type ticketSnapshot struct {
	Backlog    []string
	InProgress []string
	InReview   []string
	Done       []string
	Details    map[string]ticketstate.Ticket
}

func (s ticketSnapshot) hash() string {
	var lines []string
	names := append([]string{}, s.Backlog...)
	names = append(names, s.InProgress...)
	names = append(names, s.InReview...)
	names = append(names, s.Done...)
	for _, name := range names {
		t := s.Details[name]
		lines = append(lines, fmt.Sprintf("%s|%s|%s|%s|%s|%s", t.Status, name, t.ID, t.Blocker, strings.Join(t.BlockedBy, ","), t.NextAction))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return fmt.Sprintf("%x", sum[:8])
}

func snapshotTickets(repoPath string) (ticketSnapshot, error) {
	var snap ticketSnapshot
	snap.Details = make(map[string]ticketstate.Ticket)
	all, err := ticketstate.List(repoPath)
	if err != nil {
		return snap, err
	}
	for _, t := range all {
		snap.Details[t.Name] = t
		switch t.Status {
		case ticketstate.StatusBacklog:
			snap.Backlog = append(snap.Backlog, t.Name)
		case ticketstate.StatusInProgress:
			snap.InProgress = append(snap.InProgress, t.Name)
		case ticketstate.StatusInReview:
			snap.InReview = append(snap.InReview, t.Name)
		case ticketstate.StatusDone:
			snap.Done = append(snap.Done, t.Name)
		}
	}
	sort.Strings(snap.Backlog)
	sort.Strings(snap.InProgress)
	sort.Strings(snap.InReview)
	sort.Strings(snap.Done)
	return snap, nil
}

func listTicketFiles(repoPath, status string) ([]string, error) {
	dir := filepath.Join(repoPath, "docs", "tickets", status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s tickets: %w", status, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "README.md" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func validateEngineerTicketGate(before, after ticketSnapshot) error {
	return validateEngineerTicketGateWithEvidence("", before, after)
}

func validateEngineerTicketGateWithEvidence(repoPath string, before, after ticketSnapshot) error {
	if err := validateEngineerTicketMovement(before, after); err != nil {
		return err
	}
	if strings.TrimSpace(repoPath) == "" {
		return nil
	}
	doneAdded := setDifference(stringSet(after.Done), stringSet(before.Done))
	return validateCompletedTicketEvidence(repoPath, doneAdded)
}

func validateEngineerTicketMovement(before, after ticketSnapshot) error {
	if len(before.Backlog)+len(before.InProgress) == 0 {
		if len(after.InProgress) > 0 {
			return fmt.Errorf(
				"ticket gate: engineer created in-progress work without any open ticket at start: %s",
				strings.Join(after.InProgress, ", "),
			)
		}
		return nil
	}

	beforeIP := stringSet(before.InProgress)
	afterIP := stringSet(after.InProgress)
	inReviewAdded := setDifference(stringSet(after.InReview), stringSet(before.InReview))
	doneAdded := setDifference(stringSet(after.Done), stringSet(before.Done))

	eligibleBefore := before.eligibleInProgressSet()
	if len(eligibleBefore) > 0 {
		var newInProgress []string
		for _, name := range after.InProgress {
			if !beforeIP[name] {
				newInProgress = append(newInProgress, name)
			}
		}
		if len(newInProgress) > 0 {
			return fmt.Errorf(
				"ticket gate: engineer must drain existing in-progress tickets before claiming new work: %s",
				strings.Join(newInProgress, ", "),
			)
		}

		for name := range eligibleBefore {
			if doneAdded[name] {
				return nil
			}
			if after.ticketInStatus(name, ticketstate.StatusBacklog) && after.ticketHasBlockerNote(name) {
				return nil
			}
			if after.ticketInStatus(name, ticketstate.StatusInProgress) && after.ticketHasLinkedDependency(name) {
				return nil
			}
			if inReviewAdded[name] && after.ticketHasReviewMetadata(name) {
				return nil
			}
		}

		removed := setDifference(eligibleBefore, afterIP)
		if len(removed) == 0 {
			return fmt.Errorf(
				"ticket gate: engineer ended without completing, blocking, or returning any eligible in-progress ticket; remaining: %s",
				strings.Join(after.InProgress, ", "),
			)
		}
		return fmt.Errorf(
			"ticket gate: engineer removed in-progress ticket(s) without moving them to done or backlog with blocker metadata: %s",
			strings.Join(sortedSetKeys(removed), ", "),
		)
	}

	var newlyClaimed []string
	for _, name := range after.InProgress {
		if !beforeIP[name] {
			newlyClaimed = append(newlyClaimed, name)
		}
	}
	if len(newlyClaimed) > 0 {
		return fmt.Errorf(
			"ticket gate: engineer cannot hand off with newly claimed ticket(s) still in docs/tickets/in-progress: %s",
			strings.Join(newlyClaimed, ", "),
		)
	}
	if len(doneAdded) == 0 {
		for name := range inReviewAdded {
			if after.ticketHasReviewMetadata(name) {
				return nil
			}
		}
		if len(before.Backlog) == 0 && len(eligibleBefore) == 0 {
			return nil
		}
		return fmt.Errorf(
			"ticket gate: engineer ended without moving any ticket to docs/tickets/done while %d open ticket(s) existed",
			len(before.Backlog)+len(before.InProgress),
		)
	}
	return nil
}

func (s ticketSnapshot) eligibleInProgressSet() map[string]bool {
	out := make(map[string]bool)
	for _, name := range s.InProgress {
		t, ok := s.Details[name]
		if !ok || t.EligibleInProgress() {
			out[name] = true
		}
	}
	return out
}

func (s ticketSnapshot) ticketInStatus(name, status string) bool {
	t, ok := s.Details[name]
	if ok {
		return t.Status == status
	}
	switch status {
	case ticketstate.StatusBacklog:
		return stringSet(s.Backlog)[name]
	case ticketstate.StatusInProgress:
		return stringSet(s.InProgress)[name]
	case ticketstate.StatusInReview:
		return stringSet(s.InReview)[name]
	case ticketstate.StatusDone:
		return stringSet(s.Done)[name]
	default:
		return false
	}
}

func (s ticketSnapshot) ticketHasBlockerNote(name string) bool {
	t, ok := s.Details[name]
	if !ok {
		return false
	}
	return meaningfulTicketGateField(t.Blocker)
}

func (s ticketSnapshot) ticketHasLinkedDependency(name string) bool {
	t, ok := s.Details[name]
	if !ok || len(t.BlockedBy) == 0 {
		return false
	}
	for _, dep := range t.BlockedBy {
		if s.hasTicketReferenceExcept(dep, name) {
			return true
		}
	}
	return false
}

func (s ticketSnapshot) ticketHasReviewMetadata(name string) bool {
	t, ok := s.Details[name]
	if !ok {
		return false
	}
	owner := strings.ToLower(t.Owner)
	next := strings.ToLower(t.NextAction)
	return strings.Contains(owner, "qa") ||
		strings.Contains(owner, "review") ||
		strings.Contains(next, "review") ||
		strings.Contains(next, "approval")
}

func (s ticketSnapshot) hasTicketReferenceExcept(ref, excludedName string) bool {
	ref = strings.Trim(strings.TrimSpace(ref), `"'`)
	if ref == "" {
		return false
	}
	for name, t := range s.Details {
		if name == excludedName {
			continue
		}
		if strings.EqualFold(ref, t.ID) || strings.EqualFold(ref, name) {
			return true
		}
	}
	names := append([]string{}, s.Backlog...)
	names = append(names, s.InProgress...)
	names = append(names, s.InReview...)
	names = append(names, s.Done...)
	for _, name := range names {
		if name == excludedName {
			continue
		}
		if strings.EqualFold(ref, name) || strings.Contains(strings.ToLower(name), strings.ToLower(ref)) {
			return true
		}
	}
	return false
}

func meaningfulTicketGateField(value string) bool {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	switch strings.ToLower(value) {
	case "", "[]", "none", "null", "nil", "tbd", "todo":
		return false
	default:
		return true
	}
}

func validateCompletedTicketEvidence(repoPath string, doneAdded map[string]bool) error {
	for _, name := range sortedSetKeys(doneAdded) {
		path := filepath.Join(repoPath, "docs", "tickets", "done", name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ticket gate: read completed ticket %s: %w", name, err)
		}
		if err := validateOneCompletedTicketEvidence(name, string(data)); err != nil {
			return err
		}
	}
	return nil
}

func validateOneCompletedTicketEvidence(name, text string) error {
	frontmatter := parseTicketGateFrontmatter(text)
	workType := strings.Trim(strings.ToLower(frontmatter["work_type"]), `"'`)
	endToEndEvidence := strings.Trim(strings.ToLower(frontmatter["end_to_end_evidence"]), `"'`)
	if workType != "feature" && endToEndEvidence != "required" {
		return nil
	}
	var missing []string
	if workType != "feature" {
		missing = append(missing, "work_type: feature")
	}
	if endToEndEvidence != "required" {
		missing = append(missing, "end_to_end_evidence: required")
	}
	if frontmatterFieldEmpty(frontmatter["bdd_scenarios"]) {
		missing = append(missing, "bdd_scenarios")
	}
	if frontmatterFieldEmpty(frontmatter["evidence_links"]) {
		missing = append(missing, "evidence_links")
	}
	if frontmatterFieldEmpty(frontmatter["verified_by"]) {
		missing = append(missing, "verified_by")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"ticket gate: feature ticket %s cannot move to done without BDD scenario evidence: missing %s",
			name,
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func parseTicketGateFrontmatter(text string) map[string]string {
	fields := make(map[string]string)
	scanner := strings.Split(text, "\n")
	if len(scanner) == 0 || strings.TrimSpace(scanner[0]) != "---" {
		return fields
	}
	for _, line := range scanner[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return fields
}

func frontmatterFieldEmpty(value string) bool {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	if value == "" || value == "[]" || strings.EqualFold(value, "tbd") || strings.EqualFold(value, "none") {
		return true
	}
	return false
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func setDifference(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool)
	for value := range a {
		if !b[value] {
			out[value] = true
		}
	}
	return out
}

func sortedSetKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
