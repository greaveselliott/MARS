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
    "dedupe_key": { "type": "string", "description": "Optional stable dedupe key for machine-generated tickets." },
    "metadata":   { "type": "object", "additionalProperties": { "type": "string" }, "description": "Optional machine-readable string metadata written into frontmatter." },
    "source":     { "type": "string", "description": "Where this ticket originated (e.g. 'weekly-priorities.md — This week item 3')" },
    "depends_on": { "type": "array", "items": { "type": "string" }, "description": "Ticket IDs this depends on (e.g. ['T-001', 'T-003'])" },
    "body":       { "type": "string", "description": "Full ticket body: Context, Requirements, Affected Files, Design Guidance, Acceptance criteria sections" }
  },
  "required": ["title", "priority", "body"]
}`

type ticketCreateArgs struct {
	Title      string            `json:"title"`
	Priority   string            `json:"priority"`
	Complexity string            `json:"complexity"`
	Kind       string            `json:"kind"`
	DedupeKey  string            `json:"dedupe_key"`
	Metadata   map[string]string `json:"metadata"`
	Source     string            `json:"source"`
	DependsOn  []string          `json:"depends_on"`
	Body       string            `json:"body"`
}

// TicketInput is the shared ticket creation shape used by agents and scanner-generated backlog items.
type TicketInput struct {
	Title      string
	Priority   string
	Complexity string
	Kind       string
	DedupeKey  string
	Metadata   map[string]string
	Source     string
	DependsOn  []string
	Body       string
}

type existingTicket struct {
	ID        string
	Title     string
	Kind      string
	DedupeKey string
	Number    int
	Path      string // relative to repo root, e.g. "docs/tickets/done/T-001-foo.md"
	Status    string // "backlog", "in-progress", or "done"
}

var ticketNumberRe = regexp.MustCompile(`T-(\d+)`)

func registerTicketCreate(r *Registry) error {
	return r.Register(
		"ticket_create",
		"Create a ticket in docs/tickets/backlog/ with automatic deduplication. "+
			"If a ticket with the same topic already exists (in backlog, in-progress, or done), "+
			"the tool returns the existing ticket path instead of creating a duplicate.",
		json.RawMessage(ticketCreateSchema),
		handleTicketCreate,
	)
}

func handleTicketCreate(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("ticket_create: parse arguments: %w", err)
	}
	return CreateTicket(root, TicketInput{
		Title:      args.Title,
		Priority:   args.Priority,
		Complexity: args.Complexity,
		Kind:       args.Kind,
		DedupeKey:  args.DedupeKey,
		Metadata:   args.Metadata,
		Source:     args.Source,
		DependsOn:  args.DependsOn,
		Body:       args.Body,
	})
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
	fmt.Fprintf(&content, "%s\n", strings.TrimSpace(input.Body))

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

func scanExistingTickets(repoRoot string) ([]existingTicket, error) {
	ticketsDir := filepath.Join(repoRoot, "docs", "tickets")
	statuses := []string{"backlog", "in-progress", "done"}

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

	if err := os.WriteFile(absPath, append(data, []byte(update.String())...), 0o644); err != nil {
		return false, fmt.Errorf("ticket_create: update existing ticket: %w", err)
	}
	return true, nil
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

// isSubsetMatch returns true if the shorter word set is a subset of the longer one.
// This catches "implement scoring system" matching "implement scoring system component".
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

	matches := 0
	for _, w := range shorter {
		if longerSet[w] {
			matches++
		}
	}

	// Require all words in the shorter set to match (subset),
	// or at least 80% if the shorter set has 5+ words (fuzzy tolerance).
	threshold := len(shorter)
	if len(shorter) >= 5 {
		threshold = len(shorter) * 4 / 5
	}
	return matches >= threshold
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
