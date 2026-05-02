package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ticketSnapshot struct {
	Backlog    []string
	InProgress []string
	Done       []string
}

func snapshotTickets(repoPath string) (ticketSnapshot, error) {
	var snap ticketSnapshot
	var err error
	if snap.Backlog, err = listTicketFiles(repoPath, "backlog"); err != nil {
		return snap, err
	}
	if snap.InProgress, err = listTicketFiles(repoPath, "in-progress"); err != nil {
		return snap, err
	}
	if snap.Done, err = listTicketFiles(repoPath, "done"); err != nil {
		return snap, err
	}
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
	doneAdded := setDifference(stringSet(after.Done), stringSet(before.Done))

	if len(before.InProgress) > 0 {
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

		removed := setDifference(beforeIP, afterIP)
		if len(removed) == 0 {
			return fmt.Errorf(
				"ticket gate: engineer ended without completing any existing in-progress ticket; remaining: %s",
				strings.Join(after.InProgress, ", "),
			)
		}
		for name := range removed {
			if doneAdded[name] {
				return nil
			}
		}
		return fmt.Errorf(
			"ticket gate: engineer removed in-progress ticket(s) without moving them to done: %s",
			strings.Join(sortedSetKeys(removed), ", "),
		)
	}

	if len(after.InProgress) > 0 {
		return fmt.Errorf(
			"ticket gate: engineer cannot hand off with newly claimed ticket(s) still in docs/tickets/in-progress: %s",
			strings.Join(after.InProgress, ", "),
		)
	}
	if len(doneAdded) == 0 {
		return fmt.Errorf(
			"ticket gate: engineer ended without moving any ticket to docs/tickets/done while %d open ticket(s) existed",
			len(before.Backlog)+len(before.InProgress),
		)
	}
	return nil
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
