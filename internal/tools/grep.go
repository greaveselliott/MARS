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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const grepSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "pattern": { "type": "string", "description": "Regular expression (RE2 syntax)" },
    "glob": { "type": "string", "description": "Optional glob filter relative to repo root (default **/*)" },
    "max_matches": { "type": "integer", "description": "Maximum number of matching lines (default 1000, max 10000)" }
  },
  "required": ["pattern"]
}`

type grepArgs struct {
	Pattern    string `json:"pattern"`
	Glob       string `json:"glob"`
	MaxMatches int    `json:"max_matches"`
}

func registerGrep(r *Registry) error {
	return r.Register("grep", "Search files under the repository root for lines matching a regular expression.", json.RawMessage(grepSchema), handleGrep)
}

func handleGrep(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args grepArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("grep: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Pattern) == "" {
		return ToolResult{}, fmt.Errorf("grep: field pattern is required")
	}
	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return ToolResult{}, fmt.Errorf("grep: invalid regex: %w", err)
	}
	max := args.MaxMatches
	if max <= 0 {
		max = 1000
	}
	if max > 10000 {
		max = 10000
	}
	globPat := strings.TrimSpace(args.Glob)
	if globPat == "" {
		globPat = "**/*"
	}
	if filepath.IsAbs(globPat) {
		return ToolResult{}, fmt.Errorf("grep: glob must be relative to repository root")
	}

	matches, err := filepath.Glob(filepath.Join(root.Abs(), globPat))
	if err != nil {
		return ToolResult{}, fmt.Errorf("grep: glob %q: %w", globPat, err)
	}

	var buf bytes.Buffer
	matchCount := 0
	truncated := false
	const maxBytes = DefaultMaxToolOutputBytes
	for _, file := range matches {
		if matchCount >= max {
			break
		}
		rel, _ := filepath.Rel(root.Abs(), file)
		rel = filepath.ToSlash(rel)
		if IsGeneratedWorkspacePath(rel) {
			continue
		}
		fi, err := os.Stat(file)
		if err != nil || fi.IsDir() {
			continue
		}
		if fi.Size() > 4<<20 {
			continue
		}
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		head := make([]byte, 512)
		n, _ := f.Read(head)
		if IsProbablyBinary(head[:n]) {
			_ = f.Close()
			continue
		}
		r := io.MultiReader(bytes.NewReader(head[:n]), f)
		sc := bufio.NewScanner(r)
		scanBuf := make([]byte, 0, 64*1024)
		sc.Buffer(scanBuf, 512*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			if matchCount >= max {
				_ = f.Close()
				goto done
			}
			line := sc.Bytes()
			if !re.Match(line) {
				continue
			}
			matchCount++
			entry := fmt.Sprintf("%s:%d:%s\n", rel, lineNo, string(line))
			if buf.Len()+len(entry) > maxBytes {
				truncated = true
				_ = f.Close()
				goto done
			}
			buf.WriteString(entry)
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			return ToolResult{}, fmt.Errorf("grep: read %s: %w", rel, err)
		}
	}
done:
	if buf.Len() == 0 && !truncated {
		return ToolResult{Output: "(no matches)"}, nil
	}
	out := strings.TrimSpace(buf.String())
	if truncated && !strings.Contains(out, "truncated") {
		out += "\n(grep output truncated)"
	}
	return ToolResult{Output: out, Truncated: truncated}, nil
}
