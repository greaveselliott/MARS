/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/integrations"
)

const (
	StatusDisabled   = "disabled"
	StatusDropped    = "dropped"
	StatusCreated    = "created"
	StatusReconciled = "reconciled"
)

type Repository struct {
	ID     string
	Path   string
	Remote string
	Config integrations.Config
}

type RawIssue struct {
	Key     string
	Project string
	Data    map[string]any
}

type Issue struct {
	Key          string
	Project      string
	Summary      string
	Description  string
	URL          string
	Created      string
	Updated      string
	Priority     string
	Sprint       string
	SprintActive bool
	Rank         string
	Status       string
	Epic         string
	BlockedBy    []string
	Labels       []string
}

type MirrorResult struct {
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	JiraKey         string `json:"jira_key,omitempty"`
	RepoID          string `json:"repo_id,omitempty"`
	RepoPath        string `json:"repo_path,omitempty"`
	TicketPath      string `json:"ticket_path,omitempty"`
	Created         bool   `json:"created"`
	LLMJobsEnqueued int    `json:"llm_jobs_enqueued"`
}

func RawIssueFromWebhookPayload(data []byte) (RawIssue, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return RawIssue{}, fmt.Errorf("jira: parse webhook payload: %w", err)
	}
	if issue, ok := asMap(root["issue"]); ok {
		return rawIssueFromMap(issue)
	}
	return rawIssueFromMap(root)
}

func RawIssuesFromSearchPayload(data []byte) ([]RawIssue, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("jira: parse search payload: %w", err)
	}
	items, ok := asSlice(root["issues"])
	if !ok {
		return nil, nil
	}
	out := make([]RawIssue, 0, len(items))
	for _, item := range items {
		issueMap, ok := asMap(item)
		if !ok {
			continue
		}
		raw, err := rawIssueFromMap(issueMap)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}

func rawIssueFromMap(issue map[string]any) (RawIssue, error) {
	key := strings.TrimSpace(stringValue(issue["key"]))
	fields, _ := asMap(issue["fields"])
	project := projectKey(fields)
	if project == "" {
		project = projectFromKey(key)
	}
	if key == "" {
		return RawIssue{}, fmt.Errorf("jira: issue payload missing key")
	}
	return RawIssue{Key: key, Project: project, Data: issue}, nil
}

func (r RawIssue) Normalize(fieldsCfg integrations.JIRAFieldsConfig) Issue {
	fields, _ := asMap(r.Data["fields"])
	issue := Issue{
		Key:         strings.TrimSpace(r.Key),
		Project:     strings.TrimSpace(r.Project),
		Summary:     fieldString(fields, "summary"),
		Description: textFromAny(fields["description"]),
		URL:         stringValue(r.Data["self"]),
		Created:     fieldString(fields, "created"),
		Updated:     fieldString(fields, "updated"),
		Priority:    nestedName(fields["priority"]),
		Status:      nestedName(fields["status"]),
		Sprint:      customFieldString(fields, fieldsCfg.Sprint),
		Rank:        customFieldString(fields, fieldsCfg.Rank),
		Epic:        customFieldString(fields, fieldsCfg.EpicLink),
		BlockedBy:   blockedByKeys(fields),
		Labels:      labels(fields),
	}
	if issue.Project == "" {
		issue.Project = projectKey(fields)
	}
	if issue.Project == "" {
		issue.Project = projectFromKey(issue.Key)
	}
	if issue.Sprint == "" {
		issue.Sprint = customFieldString(fields, "sprint")
	}
	issue.SprintActive = sprintActive(fields, fieldsCfg.Sprint)
	return issue.Sanitized()
}

func (i Issue) Sanitized() Issue {
	i.Key = strings.TrimSpace(i.Key)
	i.Project = strings.TrimSpace(i.Project)
	i.Summary = redactSensitiveText(strings.TrimSpace(i.Summary))
	i.Description = redactSensitiveText(strings.TrimSpace(i.Description))
	i.URL = redactSensitiveText(strings.TrimSpace(i.URL))
	i.Created = strings.TrimSpace(i.Created)
	i.Updated = strings.TrimSpace(i.Updated)
	i.Priority = redactSensitiveText(strings.TrimSpace(i.Priority))
	i.Sprint = redactSensitiveText(strings.TrimSpace(i.Sprint))
	i.Rank = redactSensitiveText(strings.TrimSpace(i.Rank))
	i.Status = redactSensitiveText(strings.TrimSpace(i.Status))
	i.Epic = redactSensitiveText(strings.TrimSpace(i.Epic))
	i.BlockedBy = cleanStringList(i.BlockedBy)
	i.Labels = cleanStringList(i.Labels)
	if i.Summary == "" {
		i.Summary = i.Key
	}
	return i
}

func projectKey(fields map[string]any) string {
	project, ok := asMap(fields["project"])
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue(project["key"]))
}

func projectFromKey(key string) string {
	before, _, ok := strings.Cut(strings.TrimSpace(key), "-")
	if !ok {
		return ""
	}
	return before
}

func fieldString(fields map[string]any, key string) string {
	return strings.TrimSpace(stringValue(fields[key]))
}

func nestedName(v any) string {
	m, ok := asMap(v)
	if !ok {
		return strings.TrimSpace(stringValue(v))
	}
	for _, key := range []string{"name", "key", "value"} {
		if value := strings.TrimSpace(stringValue(m[key])); value != "" {
			return value
		}
	}
	return ""
}

func customFieldString(fields map[string]any, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return textFromAny(fields[id])
}

func sprintActive(fields map[string]any, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	return activeValue(fields[id])
}

func activeValue(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		state := strings.ToLower(strings.TrimSpace(stringValue(t["state"])))
		return state == "active"
	case []any:
		for _, item := range t {
			if activeValue(item) {
				return true
			}
		}
	}
	return false
}

func blockedByKeys(fields map[string]any) []string {
	if items, ok := asSlice(fields["blocked_by"]); ok {
		var out []string
		for _, item := range items {
			if value := strings.TrimSpace(stringValue(item)); value != "" {
				out = append(out, value)
			}
		}
		return cleanStringList(out)
	}
	links, ok := asSlice(fields["issuelinks"])
	if !ok {
		return nil
	}
	var out []string
	for _, link := range links {
		linkMap, ok := asMap(link)
		if !ok {
			continue
		}
		linkType, _ := asMap(linkMap["type"])
		if !strings.Contains(strings.ToLower(stringValue(linkType["name"])), "block") {
			continue
		}
		if inward, ok := asMap(linkMap["inwardIssue"]); ok {
			if key := strings.TrimSpace(stringValue(inward["key"])); key != "" {
				out = append(out, key)
			}
		}
	}
	return cleanStringList(out)
}

func labels(fields map[string]any) []string {
	items, ok := asSlice(fields["labels"])
	if !ok {
		return nil
	}
	var out []string
	for _, item := range items {
		if value := strings.TrimSpace(stringValue(item)); value != "" {
			out = append(out, value)
		}
	}
	return cleanStringList(out)
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func asSlice(v any) ([]any, bool) {
	items, ok := v.([]any)
	return items, ok
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(t)
	}
}

func textFromAny(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case []any:
		var parts []string
		for _, item := range t {
			if value := textFromAny(item); value != "" {
				parts = append(parts, value)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "name", "key", "value"} {
			if value := strings.TrimSpace(stringValue(t[key])); value != "" {
				return value
			}
		}
		if content, ok := asSlice(t["content"]); ok {
			return textFromAny(content)
		}
	}
	return strings.TrimSpace(stringValue(v))
}

func cleanStringList(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range in {
		item = redactSensitiveText(strings.TrimSpace(item))
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

var sensitivePairRE = regexp.MustCompile(`(?i)(token|secret|api[_-]?key|password|signature|sig)=([^&\s]+)`)
var bearerRE = regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9._~+/=-]+`)

func redactSensitiveText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = bearerRE.ReplaceAllString(value, `${1}[redacted]`)
	value = sensitivePairRE.ReplaceAllString(value, `${1}=[redacted]`)
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		query := parsed.Query()
		changed := false
		for key := range query {
			if looksSensitiveKey(key) {
				query.Set(key, "[redacted]")
				changed = true
			}
		}
		if changed {
			parsed.RawQuery = query.Encode()
			value = parsed.String()
		}
	}
	return value
}

func looksSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "token") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "signature") ||
		strings.Contains(key, "sig") ||
		strings.Contains(key, "password") ||
		strings.Contains(key, "api_key") ||
		strings.Contains(key, "apikey")
}
