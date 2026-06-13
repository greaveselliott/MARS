/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ticketCreateSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "title":      { "type": "string", "description": "Concise, action-oriented ticket title (e.g. 'Implement wave progression system')" },
    "priority":   { "type": "string", "enum": ["high", "medium", "low"], "description": "Ticket priority" },
    "complexity": { "type": "string", "enum": ["small", "medium", "large"], "description": "Estimated complexity" },
    "kind":       { "type": "string", "enum": ["standard", "intervention-debt"], "description": "Optional machine-readable ticket kind. Use intervention-debt for telemetry/self-improvement debt." },
    "work_type":  { "type": "string", "enum": ["feature", "enabler", "research", "docs", "intervention-debt"], "description": "Operating-model work type. Feature tickets require BDD scenario evidence before done." },
    "bdd_scenarios": { "type": "array", "items": { "type": "string" }, "description": "BDD scenario IDs this ticket implements, e.g. ['F-001-S001']." },
    "end_to_end_evidence": { "type": "string", "enum": ["required", "not_applicable"], "description": "Whether completion requires E2E/integration evidence." },
    "evidence_links": { "type": "array", "items": { "type": "string" }, "description": "Evidence links or commands proving BDD scenarios passed." },
    "verified_by": { "type": "string", "description": "Role, command, or human verifier for completion evidence." },
    "owner": { "type": "string", "description": "Current owner or role responsible for the ticket. Use TBD when unknown." },
    "last_attempt": { "type": "string", "description": "ISO date or timestamp for the last meaningful attempt. Use TBD when none." },
    "blocker": { "type": "string", "description": "Concrete blocker note, or none/TBD when unblocked." },
    "blocked_by": { "type": "array", "items": { "type": "string" }, "description": "Ticket IDs that must land before this ticket can resume." },
    "trace_id": { "type": "string", "description": "Trace ID for the current or most recent attempt. Use TBD when absent." },
    "next_action": { "type": "string", "description": "Concrete next action for resuming or unblocking the ticket." },
    "dedupe_key": { "type": "string", "description": "Optional stable dedupe key for machine-generated tickets." },
    "metadata":   { "type": "object", "additionalProperties": { "type": "string" }, "description": "Optional machine-readable string metadata written into frontmatter." },
    "source":     { "type": "string", "description": "Where this ticket originated (e.g. 'weekly-priorities.md — This week item 3')" },
    "depends_on": { "type": "array", "items": { "type": "string" }, "description": "Ticket IDs this depends on (e.g. ['T-001', 'T-003'])" },
    "body":       { "type": "string", "description": "Full ticket body: Context, Requirements, Affected Files, Design Guidance, Acceptance criteria sections" }
  },
  "required": ["title", "priority", "body"]
}`

type ticketCreateArgs struct {
	Title            string            `json:"title"`
	Priority         string            `json:"priority"`
	Complexity       string            `json:"complexity"`
	Kind             string            `json:"kind"`
	WorkType         string            `json:"work_type"`
	BDDScenarios     []string          `json:"bdd_scenarios"`
	EndToEndEvidence string            `json:"end_to_end_evidence"`
	EvidenceLinks    []string          `json:"evidence_links"`
	VerifiedBy       string            `json:"verified_by"`
	Owner            string            `json:"owner"`
	LastAttempt      string            `json:"last_attempt"`
	Blocker          string            `json:"blocker"`
	BlockedBy        []string          `json:"blocked_by"`
	TraceID          string            `json:"trace_id"`
	NextAction       string            `json:"next_action"`
	DedupeKey        string            `json:"dedupe_key"`
	Metadata         map[string]string `json:"metadata"`
	Source           string            `json:"source"`
	DependsOn        []string          `json:"depends_on"`
	Body             string            `json:"body"`
}

// TicketInput is the shared ticket creation shape used by agents and scanner-generated backlog items.
type TicketInput struct {
	Title            string
	Priority         string
	Complexity       string
	Kind             string
	WorkType         string
	BDDScenarios     []string
	EndToEndEvidence string
	EvidenceLinks    []string
	VerifiedBy       string
	Owner            string
	LastAttempt      string
	Blocker          string
	BlockedBy        []string
	TraceID          string
	NextAction       string
	DedupeKey        string
	Metadata         map[string]string
	Source           string
	DependsOn        []string
	Body             string
}

type existingTicket struct {
	ID           string
	Title        string
	Kind         string
	WorkType     string
	BDDScenarios []string
	DedupeKey    string
	Number       int
	Path         string // relative to repo root, e.g. "docs/tickets/done/T-001-foo.md"
	Status       string // "backlog", "in-progress", "in-review", or "done"
}

var ticketNumberRe = regexp.MustCompile(`T-(\d+)`)
var compactedTriageRe = regexp.MustCompile(`(?m)- ([0-9]+) earlier triage update\(s\) compacted`)

const (
	triageUpdateHeading    = "\n\n## Latest Triage Update\n\n"
	compactedTriageHeading = "\n\n## Earlier Triage Updates Compacted\n\n"
	maxLatestTriageUpdates = 3
)

func registerTicketCreate(r *Registry) error {
	return r.Register(
		"ticket_create",
		"Create a ticket in docs/tickets/backlog/ with automatic deduplication. "+
			"If a ticket with the same topic already exists (in backlog, in-progress, in-review, or done), "+
			"the tool returns the existing ticket path instead of creating a duplicate.",
		json.RawMessage(ticketCreateSchema),
		handleTicketCreate,
	)
}

func handleTicketCreate(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("ticket_create: parse arguments: %s", ticketCreateParseHint(err))
	}
	args = withInferredTicketCreateScenarios(ctx, args)
	return CreateTicket(root, TicketInput(args))
}

func withInferredTicketCreateScenarios(ctx context.Context, args ticketCreateArgs) ticketCreateArgs {
	if len(args.BDDScenarios) > 0 || isInterventionDebtTicket(args) {
		return args
	}
	if inferred := ticketCreateScenarioIDsFromArgs(args); len(inferred) > 0 {
		args.BDDScenarios = inferred
		return args
	}
	session, ok := SessionFromContext(ctx)
	if !ok {
		return args
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "cto" && role != "cto-weekly" {
		return args
	}
	if pending := pendingCTOHandoffRequiredScenarios(session); len(pending) > 0 {
		args.BDDScenarios = pending
	}
	return args
}

func ticketCreateScenarioIDsFromArgs(args ticketCreateArgs) []string {
	var b strings.Builder
	b.WriteString(args.Title)
	b.WriteByte('\n')
	b.WriteString(args.Source)
	b.WriteByte('\n')
	b.WriteString(args.Body)
	for key, value := range args.Metadata {
		b.WriteByte('\n')
		b.WriteString(key)
		b.WriteByte(' ')
		b.WriteString(value)
	}
	return orderedFeatureScenarioIDs(b.String())
}

func ticketCreateParseHint(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "bdd_scenarios") && strings.Contains(msg, "[]string") {
		return fmt.Sprintf("bdd_scenarios must be a JSON array, not a quoted string; use \"bdd_scenarios\":[\"F-001-S002\"]: %s", msg)
	}
	if strings.Contains(msg, "blocked_by") && strings.Contains(msg, "[]string") {
		return fmt.Sprintf("blocked_by must be a JSON array, not a quoted string; use \"blocked_by\":[\"T-001\"]: %s", msg)
	}
	if strings.Contains(msg, "depends_on") && strings.Contains(msg, "[]string") {
		return fmt.Sprintf("depends_on must be a JSON array, not a quoted string; use \"depends_on\":[\"T-001\"]: %s", msg)
	}
	if strings.Contains(msg, "evidence_links") && strings.Contains(msg, "[]string") {
		return fmt.Sprintf("evidence_links must be a JSON array, not a quoted string; use \"evidence_links\":[\"go test ./...\"]: %s", msg)
	}
	return msg
}

// CreateTicket creates a backlog ticket under docs/tickets/backlog with automatic dedupe.
func CreateTicket(root Root, input TicketInput) (ToolResult, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return ToolResult{}, fmt.Errorf("ticket_create: title is required")
	}
	if strings.TrimSpace(input.Body) == "" {
		return ToolResult{}, fmt.Errorf("ticket_create: body is required — include Context, Requirements, and Acceptance criteria sections")
	}

	existing, err := scanExistingTickets(root.Abs())
	if err != nil {
		return ToolResult{}, fmt.Errorf("ticket_create: scan existing tickets: %w", err)
	}

	if dup := findDuplicateByDedupe(input.DedupeKey, existing); dup != nil {
		if input.Kind == "intervention-debt" && dup.Status != "done" {
			updated, err := updateExistingTicket(root, *dup, input)
			if err != nil {
				return ToolResult{}, err
			}
			if updated {
				return ToolResult{
					Output: fmt.Sprintf("UPDATED: intervention-debt ticket %q at %s (status: %s).", dup.Title, dup.Path, dup.Status),
				}, nil
			}
			return ToolResult{
				Output: fmt.Sprintf("UNCHANGED: intervention-debt ticket %q already has source %q at %s (status: %s).", dup.Title, input.Source, dup.Path, dup.Status),
			}, nil
		}
		return ToolResult{
			Output: fmt.Sprintf("DUPLICATE: ticket %q already exists at %s (status: %s). Skipping creation.", dup.Title, dup.Path, dup.Status),
		}, nil
	}

	if input.Kind != "intervention-debt" || strings.TrimSpace(input.DedupeKey) == "" {
		if dup := findDuplicate(title, existing); dup != nil {
			return ToolResult{
				Output: fmt.Sprintf("DUPLICATE: ticket %q already exists at %s (status: %s). Skipping creation.", dup.Title, dup.Path, dup.Status),
			}, nil
		}
	}

	if dup := findDuplicateByBDDScenario(input, existing); dup != nil {
		scenarios := strings.Join(normalizedScenarioSet(input.BDDScenarios), ", ")
		return ToolResult{
			Output: fmt.Sprintf("DUPLICATE: feature ticket for BDD scenario(s) %s already exists at %s (status: %s). Add depends_on or a distinct scenario if this really needs another ticket.", scenarios, dup.Path, dup.Status),
		}, nil
	}

	nextNum := 1
	for _, t := range existing {
		if t.Number >= nextNum {
			nextNum = t.Number + 1
		}
	}

	id := fmt.Sprintf("T-%03d", nextNum)
	slug := slugify(title)
	filename := fmt.Sprintf("%s-%s.md", id, slug)
	relPath := filepath.Join("docs", "tickets", "backlog", filename)

	complexity := input.Complexity
	if complexity == "" {
		complexity = "medium"
	}
	workType := normalizeWorkType(input.Kind, input.WorkType)
	endToEndEvidence := normalizeEndToEndEvidence(workType, input.EndToEndEvidence)
	verifiedBy := strings.TrimSpace(input.VerifiedBy)
	if verifiedBy == "" {
		verifiedBy = "TBD"
	}
	owner := defaultTicketField(input.Owner, "TBD")
	lastAttempt := defaultTicketField(input.LastAttempt, "TBD")
	blocker := defaultTicketField(input.Blocker, "none")
	traceID := defaultTicketField(input.TraceID, "TBD")
	nextAction := defaultTicketField(input.NextAction, "TBD")
	source := input.Source
	if source == "" {
		source = "weekly-priorities.md"
	}

	var deps string
	if len(input.DependsOn) > 0 {
		deps = "[" + strings.Join(input.DependsOn, ", ") + "]"
	} else {
		deps = "[]"
	}

	today := time.Now().Format("2006-01-02")

	var content strings.Builder
	fmt.Fprintf(&content, "---\n")
	fmt.Fprintf(&content, "id: %s\n", id)
	fmt.Fprintf(&content, "title: %s\n", title)
	fmt.Fprintf(&content, "priority: %s\n", input.Priority)
	fmt.Fprintf(&content, "complexity: %s\n", complexity)
	fmt.Fprintf(&content, "work_type: %s\n", workType)
	fmt.Fprintf(&content, "bdd_scenarios: %s\n", yamlInlineList(input.BDDScenarios))
	fmt.Fprintf(&content, "end_to_end_evidence: %s\n", endToEndEvidence)
	fmt.Fprintf(&content, "evidence_links: %s\n", yamlInlineList(input.EvidenceLinks))
	fmt.Fprintf(&content, "verified_by: %s\n", quoteYAMLString(verifiedBy))
	fmt.Fprintf(&content, "owner: %s\n", quoteYAMLString(owner))
	fmt.Fprintf(&content, "last_attempt: %s\n", quoteYAMLString(lastAttempt))
	fmt.Fprintf(&content, "blocker: %s\n", quoteYAMLString(blocker))
	fmt.Fprintf(&content, "blocked_by: %s\n", yamlInlineList(input.BlockedBy))
	fmt.Fprintf(&content, "trace_id: %s\n", quoteYAMLString(traceID))
	fmt.Fprintf(&content, "next_action: %s\n", quoteYAMLString(nextAction))
	if kind := strings.TrimSpace(input.Kind); kind != "" && kind != "standard" {
		fmt.Fprintf(&content, "kind: %s\n", kind)
	}
	if dedupeKey := strings.TrimSpace(input.DedupeKey); dedupeKey != "" {
		fmt.Fprintf(&content, "dedupe_key: %s\n", quoteYAMLString(dedupeKey))
	}
	if len(input.Metadata) > 0 {
		fmt.Fprintf(&content, "metadata:\n")
		for _, key := range sortedMetadataKeys(input.Metadata) {
			value := strings.TrimSpace(input.Metadata[key])
			if value == "" {
				continue
			}
			fmt.Fprintf(&content, "  %s: %s\n", safeMetadataKey(key), quoteYAMLString(value))
		}
	}
	fmt.Fprintf(&content, "source: %s\n", source)
	fmt.Fprintf(&content, "created: %s\n", today)
	fmt.Fprintf(&content, "depends_on: %s\n", deps)
	fmt.Fprintf(&content, "---\n\n")
	fmt.Fprintf(&content, "# %s: %s\n\n", id, title)
	fmt.Fprintf(&content, "%s\n", sanitizeTicketBody(input.Body, id, title))

	absPath, err := root.ResolvePath(relPath)
	if err != nil {
		return ToolResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return ToolResult{}, fmt.Errorf("ticket_create: mkdir: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(content.String()), 0o644); err != nil {
		return ToolResult{}, fmt.Errorf("ticket_create: write: %w", err)
	}

	return ToolResult{
		Output: fmt.Sprintf("created ticket %s at %s", id, relPath),
	}, nil
}

func sanitizeTicketBody(body, id, title string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for len(lines) > 0 {
		trimmed := strings.TrimSpace(lines[0])
		if trimmed == "" || duplicateTicketHeading(trimmed, id, title) {
			lines = lines[1:]
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func duplicateTicketHeading(line, id, title string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
		return false
	}
	heading := strings.TrimSpace(strings.TrimPrefix(line, "# "))
	if strings.EqualFold(heading, title) || strings.EqualFold(heading, id+": "+title) {
		return true
	}
	if prefix, rest, ok := strings.Cut(heading, ":"); ok && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(prefix)), "T-") {
		return strings.EqualFold(strings.TrimSpace(rest), title)
	}
	return false
}

func normalizeWorkType(kind, workType string) string {
	normalized := strings.TrimSpace(workType)
	if normalized != "" {
		return normalized
	}
	if strings.TrimSpace(kind) == "intervention-debt" {
		return "intervention-debt"
	}
	return "feature"
}

func normalizeEndToEndEvidence(workType, value string) string {
	normalized := strings.TrimSpace(value)
	if normalized != "" {
		return normalized
	}
	if workType == "feature" {
		return "required"
	}
	return "not_applicable"
}

func defaultTicketField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func yamlInlineList(values []string) string {
	var cleaned []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, quoteYAMLString(value))
	}
	if len(cleaned) == 0 {
		return "[]"
	}
	return "[" + strings.Join(cleaned, ", ") + "]"
}

func scanExistingTickets(repoRoot string) ([]existingTicket, error) {
	ticketsDir := filepath.Join(repoRoot, "docs", "tickets")
	statuses := []string{"backlog", "in-progress", "in-review", "done"}

	var tickets []existingTicket
	for _, status := range statuses {
		dir := filepath.Join(ticketsDir, status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" {
				continue
			}
			t := existingTicket{
				Path:   filepath.Join("docs", "tickets", status, e.Name()),
				Status: status,
			}

			if m := ticketNumberRe.FindStringSubmatch(e.Name()); len(m) == 2 {
				t.Number, _ = strconv.Atoi(m[1])
				t.ID = "T-" + m[1]
			}

			frontmatter := readTicketFrontmatter(filepath.Join(dir, e.Name()))
			if frontmatter["title"] != "" {
				t.Title = frontmatter["title"]
			} else {
				t.Title = titleFromFilename(e.Name())
			}
			t.Kind = frontmatter["kind"]
			t.WorkType = frontmatter["work_type"]
			t.BDDScenarios = parseYAMLInlineList(frontmatter["bdd_scenarios"])
			t.DedupeKey = frontmatter["dedupe_key"]

			tickets = append(tickets, t)
		}
	}
	return tickets, nil
}

func readTicketTitle(path string) string {
	return readTicketFrontmatter(path)["title"]
}

func readTicketFrontmatter(path string) map[string]string {
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
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if !inFrontmatter || strings.HasPrefix(line, " ") || !strings.Contains(line, ":") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = unquoteYAMLString(strings.TrimSpace(value))
	}
	return out
}

func parseYAMLInlineList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return nil
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		part = unquoteYAMLString(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func titleFromFilename(name string) string {
	name = strings.TrimSuffix(name, ".md")
	if m := ticketNumberRe.FindStringIndex(name); m != nil {
		name = name[m[1]:]
	}
	name = strings.TrimPrefix(name, "-")
	return strings.ReplaceAll(name, "-", " ")
}

// findDuplicate checks if a proposed title matches any existing ticket.
// Matching is case-insensitive and normalizes both titles to keyword sets,
// then checks if one is a subset of the other (handles "implement wave progression"
// matching "implement wave progression system").
func findDuplicate(proposed string, existing []existingTicket) *existingTicket {
	proposedWords := normalizeToWords(proposed)
	if len(proposedWords) == 0 {
		return nil
	}

	for i := range existing {
		existingWords := normalizeToWords(existing[i].Title)
		if len(existingWords) == 0 {
			continue
		}
		if isSubsetMatch(proposedWords, existingWords) {
			return &existing[i]
		}
	}
	return nil
}

func findDuplicateByDedupe(dedupeKey string, existing []existingTicket) *existingTicket {
	dedupeKey = strings.TrimSpace(dedupeKey)
	if dedupeKey == "" {
		return nil
	}
	for i := range existing {
		if strings.TrimSpace(existing[i].DedupeKey) == dedupeKey {
			return &existing[i]
		}
	}
	return nil
}

func findDuplicateByBDDScenario(input TicketInput, existing []existingTicket) *existingTicket {
	if strings.TrimSpace(input.Kind) == "intervention-debt" || len(input.DependsOn) > 0 {
		return nil
	}
	if normalizeWorkType(input.Kind, input.WorkType) != "feature" {
		return nil
	}
	proposed := normalizedScenarioSet(input.BDDScenarios)
	if len(proposed) == 0 {
		return nil
	}
	proposedKey := strings.Join(proposed, "\x00")
	for i := range existing {
		if existing[i].Kind == "intervention-debt" {
			continue
		}
		existingWorkType := strings.TrimSpace(existing[i].WorkType)
		if existingWorkType != "" && existingWorkType != "feature" {
			continue
		}
		existingScenarios := normalizedScenarioSet(existing[i].BDDScenarios)
		if len(existingScenarios) == 0 {
			continue
		}
		if strings.Join(existingScenarios, "\x00") == proposedKey {
			return &existing[i]
		}
		if existing[i].Status != "done" && scenarioSetsOverlap(proposed, existingScenarios) {
			return &existing[i]
		}
	}
	return nil
}

func normalizedScenarioSet(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = true
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func scenarioSetsOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := map[string]bool{}
	for _, value := range a {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = true
		}
	}
	for _, value := range b {
		value = strings.TrimSpace(value)
		if value != "" && set[value] {
			return true
		}
	}
	return false
}

func updateExistingTicket(root Root, existing existingTicket, input TicketInput) (bool, error) {
	source := strings.TrimSpace(input.Source)
	absPath, err := root.ResolvePath(existing.Path)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false, fmt.Errorf("ticket_create: read existing ticket: %w", err)
	}
	if source != "" && strings.Contains(string(data), "source: "+source) {
		return false, nil
	}

	var update strings.Builder
	fmt.Fprintf(&update, "\n\n## Latest Triage Update\n\n")
	fmt.Fprintf(&update, "- %s", time.Now().Format("2006-01-02"))
	if source != "" {
		fmt.Fprintf(&update, " — source: %s", source)
	}
	if input.Priority != "" {
		fmt.Fprintf(&update, "; priority: %s", input.Priority)
	}
	if input.DedupeKey != "" {
		fmt.Fprintf(&update, "; dedupe_key: `%s`", input.DedupeKey)
	}
	fmt.Fprintf(&update, "\n")

	if len(input.Metadata) > 0 {
		for _, key := range sortedMetadataKeys(input.Metadata) {
			value := strings.TrimSpace(input.Metadata[key])
			if value == "" {
				continue
			}
			fmt.Fprintf(&update, "- %s: %s\n", safeMetadataKey(key), value)
		}
	}

	updated := compactTriageUpdates(string(data) + update.String())
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("ticket_create: update existing ticket: %w", err)
	}
	return true, nil
}

func compactTriageUpdates(text string) string {
	parts := strings.Split(text, triageUpdateHeading)
	if len(parts) <= maxLatestTriageUpdates+1 {
		return text
	}
	base, previouslyCompacted := stripCompactedTriageMarker(parts[0])
	updates := parts[1:]
	compactCount := previouslyCompacted + len(updates) - maxLatestTriageUpdates
	recent := updates[len(updates)-maxLatestTriageUpdates:]

	var b strings.Builder
	b.WriteString(strings.TrimRight(base, "\n"))
	fmt.Fprintf(&b, "%s- %d earlier triage update(s) compacted to keep this intervention-debt ticket bounded.\n", compactedTriageHeading, compactCount)
	for _, update := range recent {
		b.WriteString(triageUpdateHeading)
		b.WriteString(update)
	}
	return b.String()
}

func stripCompactedTriageMarker(text string) (string, int) {
	idx := strings.LastIndex(text, compactedTriageHeading)
	if idx < 0 {
		return text, 0
	}
	marker := text[idx:]
	count := 0
	if m := compactedTriageRe.FindStringSubmatch(marker); len(m) == 2 {
		count, _ = strconv.Atoi(m[1])
	}
	return strings.TrimRight(text[:idx], "\n"), count
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"for": true, "in": true, "of": true, "to": true, "with": true,
}

func normalizeToWords(s string) []string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return ' '
	}, s)

	var words []string
	for _, w := range strings.Fields(s) {
		if !stopWords[w] && len(w) > 1 {
			words = append(words, w)
		}
	}
	return words
}

// isSubsetMatch returns true when every word in the shorter normalized title
// appears in the longer one (true keyword subset). The prior 80% fuzzy tolerance
// for 5+ word titles falsely merged distinct API endpoint tickets and sibling
// enabler titles that share archetype suffix words (T-030 wedge).
func isSubsetMatch(a, b []string) bool {
	shorter, longer := a, b
	if len(a) > len(b) {
		shorter, longer = b, a
	}
	if len(shorter) == 0 {
		return false
	}

	longerSet := make(map[string]bool, len(longer))
	for _, w := range longer {
		longerSet[w] = true
	}

	for _, w := range shorter {
		if !longerSet[w] {
			return false
		}
	}
	return true
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)

	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func sortedMetadataKeys(metadata map[string]string) []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func safeMetadataKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "field"
	}
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		return "field"
	}
	return out
}

func quoteYAMLString(value string) string {
	return strconv.Quote(value)
}

func unquoteYAMLString(value string) string {
	if value == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	return strings.Trim(value, `'"`)
}
