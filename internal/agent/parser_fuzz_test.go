/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/source-quality-gates.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

// FuzzToolCallsFromAssistantMessage hardens the hostile-model-output parser:
// arbitrary assistant text must never panic, and every successfully parsed
// tool call must carry valid JSON arguments and a non-empty name (T-025).
func FuzzToolCallsFromAssistantMessage(f *testing.F) {
	seeds := []string{
		"",
		"plain prose with no tool call",
		`{"name":"file_read","arguments":{"path":"README.md"}}`,
		`[{"name":"shell_exec","arguments":{"argv":["go","test","./..."]}}]`,
		`{"name":"shell_exec","arguments":{"argv":"['go', 'test', './...']"}}`,
		"```json\n{\"name\":\"file_write\",\"arguments\":{\"path\":\"a.go\",\"content\":\"x\"}}\n```",
		`<function=file_read><parameter=path>README.md</parameter></function>`,
		`<tool_call>shell_exec{"argv": ["ls"]}</tool_call>`,
		`<tool_call>file_write{path: <|"|>main.go<|"|>, content: <|"|>package main<|"|>}</tool_call>`,
		`{"name":"x","arguments":{"flag": True, "other": None}}`,
		`<tool_call>broken{</tool_call>`,
		`[{"name":"a"},{"name":"b","arguments":{"nested":{"list":[1,2,3]}}}]`,
		"{\"name\":\"x\",\"arguments\":\"{\\\"k\\\":\\\"v\\\"}\"}",
		`<function=><parameter=p>v</parameter></function>`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	for _, name := range []string{"assistant_fence.txt", "recorded_llama.txt"} {
		if data, err := os.ReadFile(filepath.Join("testdata", name)); err == nil {
			f.Add(string(data))
		}
	}

	f.Fuzz(func(t *testing.T, content string) {
		calls, err := ToolCallsFromAssistantMessage(llm.Message{Role: "assistant", Content: content})
		if err != nil {
			return
		}
		for _, call := range calls {
			if call.Function.Name == "" {
				t.Fatalf("parsed tool call with empty name from %q", content)
			}
			args := call.Function.Arguments
			if args == "" {
				continue
			}
			if !json.Valid([]byte(args)) {
				t.Fatalf("parsed tool call %q with invalid JSON arguments %q from %q",
					call.Function.Name, args, content)
			}
		}
	})
}
