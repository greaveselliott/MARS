/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const toolCreateSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "name": { "type": "string", "description": "New tool name in snake_case, e.g. cli_reference" },
    "description": { "type": "string", "description": "One-sentence tool description for the LLM tool registry" },
    "fields": {
      "type": "array",
      "description": "JSON object fields for the tool input schema",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "name": { "type": "string", "description": "Field name in snake_case" },
          "type": { "type": "string", "enum": ["string", "integer", "number", "boolean", "object", "array"], "description": "JSON Schema primitive type" },
          "description": { "type": "string", "description": "Field description for the schema" },
          "required": { "type": "boolean", "description": "Whether this field is required" }
        },
        "required": ["name", "type", "description"]
      }
    }
  },
  "required": ["name", "description"]
}`

type toolCreateArgs struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Fields      []toolCreateField `json:"fields"`
}

type toolCreateField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func registerToolCreate(r *Registry) error {
	return r.Register(
		"tool_create",
		"Scaffold a new built-in Go tool under internal/tools with schema, handler placeholder, registration reminder, and tests.",
		json.RawMessage(toolCreateSchema),
		handleToolCreate,
	)
}

func handleToolCreate(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args toolCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("tool_create: parse arguments: %w", err)
	}
	spec, err := normalizeToolSpec(args)
	if err != nil {
		return ToolResult{}, err
	}
	if err := ensureHarnessToolRoot(root); err != nil {
		return ToolResult{}, err
	}

	toolPath := filepath.Join("internal", "tools", spec.FileBase+".go")
	testPath := filepath.Join("internal", "tools", spec.FileBase+"_test.go")
	registerPath := filepath.Join("internal", "tools", "register_default.go")

	for _, rel := range []string{toolPath, testPath} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return ToolResult{}, err
		}
		if _, err := os.Stat(abs); err == nil {
			return ToolResult{}, fmt.Errorf("tool_create: %s already exists; refusing to overwrite", rel)
		} else if !os.IsNotExist(err) {
			return ToolResult{}, fmt.Errorf("tool_create: stat %s: %w", rel, err)
		}
	}

	if err := writeFileUnderRoot(root, toolPath, renderToolFile(spec)); err != nil {
		return ToolResult{}, err
	}
	if err := writeFileUnderRoot(root, testPath, renderToolTestFile(spec)); err != nil {
		return ToolResult{}, err
	}

	return ToolResult{Output: fmt.Sprintf(
		"created %s and %s\nnext: add register%s to %s, implement handle%s, extend docs/design-docs/tools-glossary.md, then run go test ./internal/tools",
		toolPath,
		testPath,
		spec.TypeName,
		registerPath,
		spec.TypeName,
	)}, nil
}

type normalizedToolSpec struct {
	Name        string
	Description string
	FileBase    string
	ConstName   string
	TypeName    string
	Fields      []normalizedToolField
}

type normalizedToolField struct {
	Name        string
	Type        string
	Description string
	Required    bool
	GoName      string
	GoType      string
	JSONTag     string
}

var toolNameRE = regexp.MustCompile(`^[a-z](?:[a-z0-9_]*[a-z0-9])?$`)

func normalizeToolSpec(args toolCreateArgs) (normalizedToolSpec, error) {
	name := strings.TrimSpace(args.Name)
	if !toolNameRE.MatchString(name) {
		return normalizedToolSpec{}, fmt.Errorf("tool_create: name must be snake_case and match %s", toolNameRE.String())
	}
	description := strings.TrimSpace(args.Description)
	if description == "" {
		return normalizedToolSpec{}, fmt.Errorf("tool_create: description is required")
	}

	spec := normalizedToolSpec{
		Name:        name,
		Description: description,
		FileBase:    name,
		TypeName:    exportedName(name),
	}
	spec.ConstName = lowerFirst(spec.TypeName)

	seen := map[string]struct{}{}
	for _, field := range args.Fields {
		nf, err := normalizeToolField(field)
		if err != nil {
			return normalizedToolSpec{}, err
		}
		if _, ok := seen[nf.Name]; ok {
			return normalizedToolSpec{}, fmt.Errorf("tool_create: duplicate field %q", nf.Name)
		}
		seen[nf.Name] = struct{}{}
		spec.Fields = append(spec.Fields, nf)
	}
	sort.Slice(spec.Fields, func(i, j int) bool {
		return spec.Fields[i].Name < spec.Fields[j].Name
	})
	return spec, nil
}

func normalizeToolField(field toolCreateField) (normalizedToolField, error) {
	name := strings.TrimSpace(field.Name)
	if !toolNameRE.MatchString(name) {
		return normalizedToolField{}, fmt.Errorf("tool_create: field name %q must be snake_case", field.Name)
	}
	typ := strings.TrimSpace(field.Type)
	allowed := []string{"string", "integer", "number", "boolean", "object", "array"}
	if !slices.Contains(allowed, typ) {
		return normalizedToolField{}, fmt.Errorf("tool_create: field %q has unsupported type %q", name, typ)
	}
	description := strings.TrimSpace(field.Description)
	if description == "" {
		return normalizedToolField{}, fmt.Errorf("tool_create: field %q description is required", name)
	}
	return normalizedToolField{
		Name:        name,
		Type:        typ,
		Description: description,
		Required:    field.Required,
		GoName:      exportedName(name),
		GoType:      goTypeForJSONType(typ),
		JSONTag:     name,
	}, nil
}

func ensureHarnessToolRoot(root Root) error {
	for _, rel := range []string{
		filepath.Join("internal", "tools", "registry.go"),
		filepath.Join("internal", "tools", "register_default.go"),
		"go.mod",
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return err
		}
		if _, err := os.Stat(abs); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("tool_create: %s is missing; this tool only scaffolds mars built-in tools", rel)
			}
			return fmt.Errorf("tool_create: stat %s: %w", rel, err)
		}
	}
	return nil
}

func writeFileUnderRoot(root Root, rel, content string) error {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("tool_create: mkdir %s: %w", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return fmt.Errorf("tool_create: write %s: %w", rel, err)
	}
	return nil
}

func renderToolFile(spec normalizedToolSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package tools\n\n")
	fmt.Fprintf(&b, "import (\n")
	fmt.Fprintf(&b, "\t\"context\"\n")
	fmt.Fprintf(&b, "\t\"encoding/json\"\n")
	fmt.Fprintf(&b, "\t\"fmt\"\n")
	fmt.Fprintf(&b, ")\n\n")
	fmt.Fprintf(&b, "const %sSchema = `{\n", spec.ConstName)
	fmt.Fprintf(&b, "  \"type\": \"object\",\n")
	fmt.Fprintf(&b, "  \"additionalProperties\": false,\n")
	fmt.Fprintf(&b, "  \"properties\": {")
	if len(spec.Fields) > 0 {
		fmt.Fprintf(&b, "\n")
		for i, field := range spec.Fields {
			comma := ","
			if i == len(spec.Fields)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "    %s: { \"type\": %s, \"description\": %s }%s\n",
				strconv.Quote(field.Name),
				strconv.Quote(field.Type),
				strconv.Quote(field.Description),
				comma,
			)
		}
		fmt.Fprintf(&b, "  },\n")
	} else {
		fmt.Fprintf(&b, "},\n")
	}
	required := requiredFieldNames(spec.Fields)
	fmt.Fprintf(&b, "  \"required\": %s\n", jsonStringArray(required))
	fmt.Fprintf(&b, "}`\n\n")
	fmt.Fprintf(&b, "type %sArgs struct {\n", lowerFirst(spec.TypeName))
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s %s `json:%q`\n", field.GoName, field.GoType, field.JSONTag)
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func register%s(r *Registry) error {\n", spec.TypeName)
	fmt.Fprintf(&b, "\treturn r.Register(%q, %q, json.RawMessage(%sSchema), handle%s)\n", spec.Name, spec.Description, spec.ConstName, spec.TypeName)
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func handle%s(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {\n", spec.TypeName)
	fmt.Fprintf(&b, "\tvar args %sArgs\n", lowerFirst(spec.TypeName))
	fmt.Fprintf(&b, "\tif err := json.Unmarshal(raw, &args); err != nil {\n")
	fmt.Fprintf(&b, "\t\treturn ToolResult{}, fmt.Errorf(%q, err)\n", spec.Name+": parse arguments: %w")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\t_ = root\n")
	fmt.Fprintf(&b, "\t_ = args\n")
	fmt.Fprintf(&b, "\treturn ToolResult{}, fmt.Errorf(%q)\n", spec.Name+": handler not implemented yet")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

func renderToolTestFile(spec normalizedToolSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package tools\n\n")
	fmt.Fprintf(&b, "import (\n")
	fmt.Fprintf(&b, "\t\"context\"\n")
	fmt.Fprintf(&b, "\t\"testing\"\n\n")
	fmt.Fprintf(&b, "\t\"github.com/stretchr/testify/require\"\n")
	fmt.Fprintf(&b, ")\n\n")
	fmt.Fprintf(&b, "func Test%s_scaffoldRequiresImplementation(t *testing.T) {\n", spec.TypeName)
	fmt.Fprintf(&b, "\tt.Parallel()\n")
	fmt.Fprintf(&b, "\troot, err := NewRoot(t.TempDir())\n")
	fmt.Fprintf(&b, "\trequire.NoError(t, err)\n")
	fmt.Fprintf(&b, "\t_, err = handle%s(context.Background(), root, []byte(`{}`))\n", spec.TypeName)
	fmt.Fprintf(&b, "\trequire.Error(t, err)\n")
	fmt.Fprintf(&b, "\trequire.Contains(t, err.Error(), %q)\n", "handler not implemented yet")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

func requiredFieldNames(fields []normalizedToolField) []string {
	var out []string
	for _, field := range fields {
		if field.Required {
			out = append(out, field.Name)
		}
	}
	return out
}

func jsonStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func exportedName(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func goTypeForJSONType(typ string) string {
	switch typ {
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "object", "array":
		return "json.RawMessage"
	default:
		return "string"
	}
}
