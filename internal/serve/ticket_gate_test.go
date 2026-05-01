package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEngineerTicketGate_allowsCompletedTicket(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		Done: []string{"T-000-setup.md", "T-001-fix-build.md"},
	}

	if err := validateEngineerTicketGate(before, after); err != nil {
		t.Fatalf("expected completed ticket to pass gate, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksInProgressHandoff(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
	}
	after := ticketSnapshot{
		InProgress: []string{"T-001-fix-build.md", "T-002-auth.md"},
		Done:       []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block in-progress tickets")
	}
	if !strings.Contains(err.Error(), "cannot hand off") {
		t.Fatalf("expected handoff error, got %v", err)
	}
	if !strings.Contains(err.Error(), "T-002-auth.md") {
		t.Fatalf("expected ticket names in error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_blocksNoCompletionWithOpenWork(t *testing.T) {
	before := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}
	after := ticketSnapshot{
		Backlog: []string{"T-001-fix-build.md"},
		Done:    []string{"T-000-setup.md"},
	}

	err := validateEngineerTicketGate(before, after)
	if err == nil {
		t.Fatal("expected gate to block no-progress completion")
	}
	if !strings.Contains(err.Error(), "without moving any ticket") {
		t.Fatalf("expected no-completion error, got %v", err)
	}
}

func TestValidateEngineerTicketGate_allowsNoOpenWork(t *testing.T) {
	if err := validateEngineerTicketGate(ticketSnapshot{}, ticketSnapshot{}); err != nil {
		t.Fatalf("expected no open work to pass gate, got %v", err)
	}
}

func TestSnapshotTickets_listsMarkdownTicketsOnly(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"backlog", "in-progress", "done"} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeTicketGateFile(t, dir, "backlog", "T-002-beta.md")
	writeTicketGateFile(t, dir, "backlog", "T-001-alpha.md")
	writeTicketGateFile(t, dir, "backlog", "README.md")
	writeTicketGateFile(t, dir, "backlog", "notes.txt")
	writeTicketGateFile(t, dir, "in-progress", "T-003-gamma.md")

	snap, err := snapshotTickets(dir)
	if err != nil {
		t.Fatalf("snapshotTickets: %v", err)
	}
	if got := strings.Join(snap.Backlog, ","); got != "T-001-alpha.md,T-002-beta.md" {
		t.Fatalf("unexpected backlog files: %s", got)
	}
	if got := strings.Join(snap.InProgress, ","); got != "T-003-gamma.md" {
		t.Fatalf("unexpected in-progress files: %s", got)
	}
}

func writeTicketGateFile(t *testing.T, repoPath, status, name string) {
	t.Helper()
	path := filepath.Join(repoPath, "docs", "tickets", status, name)
	if err := os.WriteFile(path, []byte("# ticket\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
