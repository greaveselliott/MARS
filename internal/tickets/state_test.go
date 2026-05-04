/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
package tickets

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEligibleInProgressSkipsBlockedTickets(t *testing.T) {
	t.Parallel()
	dir := setupTicketRepo(t)
	writeTicketFile(t, dir, StatusInProgress, "T-002-blocked.md", `---
id: T-002
title: Blocked
blocker: "missing dependency"
blocked_by: ["T-003"]
---

# Blocked
`)
	writeTicketFile(t, dir, StatusInProgress, "T-001-ready.md", `---
id: T-001
title: Ready
blocker: TBD
blocked_by: []
---

# Ready
`)

	tickets, err := EligibleInProgress(dir)
	if err != nil {
		t.Fatalf("EligibleInProgress: %v", err)
	}
	if len(tickets) != 1 || tickets[0].ID != "T-001" {
		t.Fatalf("expected only T-001 eligible, got %#v", tickets)
	}
}

func TestStaleInProgressUsesLastAttemptAndSkipsBlocked(t *testing.T) {
	t.Parallel()
	dir := setupTicketRepo(t)
	writeTicketFile(t, dir, StatusInProgress, "T-001-stale.md", `---
id: T-001
title: Stale
last_attempt: "2026-04-01"
---

# Stale
`)
	writeTicketFile(t, dir, StatusInProgress, "T-002-recent.md", `---
id: T-002
title: Recent
last_attempt: "2026-05-02"
---

# Recent
`)
	writeTicketFile(t, dir, StatusInProgress, "T-003-blocked.md", `---
id: T-003
title: Blocked
last_attempt: "2026-04-01"
blocker: "waiting for T-004"
blocked_by: ["T-004"]
---

# Blocked
`)

	stale, err := StaleInProgress(dir, time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC), 7*24*time.Hour)
	if err != nil {
		t.Fatalf("StaleInProgress: %v", err)
	}
	if len(stale) != 1 || stale[0].ID != "T-001" {
		t.Fatalf("expected only T-001 stale, got %#v", stale)
	}
}

func setupTicketRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, status := range []string{StatusBacklog, StatusInProgress, StatusDone} {
		if err := os.MkdirAll(filepath.Join(dir, "docs", "tickets", status), 0o755); err != nil {
			t.Fatalf("mkdir tickets: %v", err)
		}
	}
	return dir
}

func writeTicketFile(t *testing.T, repoRoot, status, name, content string) {
	t.Helper()
	path := filepath.Join(repoRoot, "docs", "tickets", status, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}
}
