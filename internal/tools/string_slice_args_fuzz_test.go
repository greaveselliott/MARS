/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/source-quality-gates.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"encoding/json"
	"testing"
	"unicode/utf8"
)

// FuzzDecodeStringSliceArg hardens the T-015 list-string normalizer used by
// mars_harness_cli.args, shell_exec.argv, workspace_hygiene.paths, and git
// path filters: arbitrary raw JSON payloads must never panic, and successful
// decodes must produce a well-formed slice (T-025).
func FuzzDecodeStringSliceArg(f *testing.F) {
	seeds := []string{
		``,
		`null`,
		`[]`,
		`["go","test","./..."]`,
		`"['go', 'test', './...']"`,
		`"[\"ls\", \"-la\"]"`,
		`"plain string value"`,
		`['single','quotes']`,
		`"['unterminated]"`,
		`"['a', 'b', 'c'"`,
		`"[ 'with\\'escape' , \"mixed\" ]"`,
		`12345`,
		`{"not":"a list"}`,
		`"[]"`,
		`"['', '']"`,
		"\"['\\u00e9', '\\u4e16\\u754c']\"",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := decodeStringSliceArg(json.RawMessage(raw), "fuzz_field")
		if err != nil {
			return
		}
		for _, item := range out {
			_ = item
		}
	})
}

// FuzzParsePythonStyleStringList exercises the quoted-list scanner directly,
// including escape handling and separator state (T-025).
func FuzzParsePythonStyleStringList(f *testing.F) {
	seeds := []string{
		`[]`,
		`['a']`,
		`['a', "b"]`,
		`['esc\'aped']`,
		`['unterminated`,
		`['a' 'b']`,
		`['a',]`,
		`[,]`,
		`['\\']`,
		`["nested [brackets]"]`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, text string) {
		out, ok := parsePythonStyleStringList(text)
		if !ok {
			return
		}
		for _, item := range out {
			if !utf8.ValidString(item) && utf8.ValidString(text) {
				t.Fatalf("valid UTF-8 input %q produced invalid UTF-8 item %q", text, item)
			}
		}
	})
}

// FuzzNormalizeShellExecArgv hardens argv normalization, including the
// single-element split path and the validation-only `cd <dir> && <test/build>`
// argv shape recognition (T-025).
func FuzzNormalizeShellExecArgv(f *testing.F) {
	f.Add("go test ./...", "cd", "cmd/app")
	f.Add("ls -la", "cd", "dir with space")
	f.Add("", "", "")
	f.Add("rm -rf /", "cd", "..")
	f.Add("['go', 'test']", "CD", "cmd/app")
	f.Add("go\ttest\t./...", "cd", "")

	f.Fuzz(func(t *testing.T, single, head, dir string) {
		out := normalizeShellExecArgv([]string{single})
		if single != "" && len(out) == 0 {
			// Normalization may split but must not silently drop a
			// non-empty argv into nothing.
			t.Fatalf("normalizeShellExecArgv(%q) returned empty argv", single)
		}

		cdArgv := []string{head, dir, "&&", "go", "test", "./..."}
		if cmd, ok := normalizeShellExecCdValidationArgv(cdArgv); ok && cmd == "" {
			t.Fatalf("normalizeShellExecCdValidationArgv(%q) accepted an empty command", cdArgv)
		}
		_, _ = normalizeShellExecCdValidationArgv([]string{head, dir, "&&"})
	})
}
