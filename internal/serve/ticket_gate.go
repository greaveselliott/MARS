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
	if len(after.InProgress) > 0 {
		return fmt.Errorf(
			"ticket gate: engineer cannot hand off while %d ticket(s) remain in docs/tickets/in-progress: %s",
			len(after.InProgress),
			strings.Join(after.InProgress, ", "),
		)
	}
	if len(before.Backlog)+len(before.InProgress) == 0 {
		return nil
	}
	if len(after.Done) <= len(before.Done) {
		return fmt.Errorf(
			"ticket gate: engineer ended without moving any ticket to docs/tickets/done while %d open ticket(s) existed",
			len(before.Backlog)+len(before.InProgress),
		)
	}
	return nil
}
