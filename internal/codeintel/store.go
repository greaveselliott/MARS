/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package codeintel

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const ParserVersion = "codeintel-v1"

type Freshness string

const (
	FreshnessFresh   Freshness = "fresh"
	FreshnessStale   Freshness = "stale"
	FreshnessPartial Freshness = "partial"
	FreshnessMissing Freshness = "missing"
)

type Store struct {
	db       *sql.DB
	repoRoot string
}

type IndexOptions struct {
	Full     bool
	MaxFiles int
}

type IndexResult struct {
	Status       Freshness `json:"status"`
	FilesSeen    int       `json:"files_seen"`
	FilesIndexed int       `json:"files_indexed"`
	FilesRemoved int       `json:"files_removed"`
	Symbols      int       `json:"symbols"`
	Edges        int       `json:"edges"`
	DurationMS   int64     `json:"duration_ms"`
	Message      string    `json:"message,omitempty"`
}

type Status struct {
	Status      Freshness `json:"status"`
	Files       int       `json:"files"`
	StaleFiles  int       `json:"stale_files"`
	NewFiles    int       `json:"new_files,omitempty"`
	Symbols     int       `json:"symbols"`
	Edges       int       `json:"edges"`
	LastRunUnix int64     `json:"last_run_unix,omitempty"`
	Message     string    `json:"message,omitempty"`
}

type Symbol struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name"`
	Kind          string `json:"kind"`
	Language      string `json:"language"`
	Path          string `json:"path"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	Signature     string `json:"signature,omitempty"`
	Freshness     string `json:"freshness,omitempty"`
	Score         int    `json:"score,omitempty"`
}

type SearchOptions struct {
	Query    string
	Kind     string
	Language string
	Path     string
	Limit    int
}

type SearchResult struct {
	Status  Freshness `json:"status"`
	Results []Symbol  `json:"results"`
	Message string    `json:"message,omitempty"`
}

type SnippetResult struct {
	Status  Freshness `json:"status"`
	Symbol  Symbol    `json:"symbol"`
	Source  string    `json:"source"`
	Message string    `json:"message,omitempty"`
}

type TraceEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

type TraceResult struct {
	Status  Freshness   `json:"status"`
	Symbol  Symbol      `json:"symbol"`
	Edges   []TraceEdge `json:"edges"`
	Message string      `json:"message,omitempty"`
}

type ImpactResult struct {
	Status       Freshness `json:"status"`
	ChangedPaths []string  `json:"changed_paths"`
	Symbols      []Symbol  `json:"symbols"`
	Tests        []string  `json:"tests"`
	Docs         []string  `json:"docs"`
	Features     []string  `json:"features"`
	Tickets      []string  `json:"tickets"`
	Message      string    `json:"message,omitempty"`
}

type fileInfo struct {
	Path     string
	Language string
	Hash     string
	ModUnix  int64
	Size     int64
}

type extracted struct {
	Symbols []Symbol
	Edges   []TraceEdge
	Text    string
}

func Open(repoRoot, dbPath string) (*Store, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil, fmt.Errorf("codeintel: repo root is required")
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("codeintel: resolve repo root: %w", err)
	}
	if eval, err := filepath.EvalSymlinks(abs); err == nil {
		abs = eval
	}
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		dbPath = DefaultDBPath(abs)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("codeintel: create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("codeintel: open sqlite %q: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, repoRoot: abs}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func DefaultDBPath(repoAbs string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	slug := filepath.Base(filepath.Clean(repoAbs))
	slug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, slug)
	if strings.Trim(slug, "-_.") == "" {
		slug = "repo"
	}
	return filepath.Join(home, ".mars-harness", "db", slug, "mars.db")
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS codeintel_files (
  path TEXT PRIMARY KEY,
  language TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  mtime_unix INTEGER NOT NULL,
  size INTEGER NOT NULL,
  parser_version TEXT NOT NULL,
  indexed_at INTEGER NOT NULL,
  status TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS codeintel_symbols (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL,
  language TEXT NOT NULL,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  qualified_name TEXT NOT NULL,
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  signature TEXT NOT NULL,
  search_text TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_codeintel_symbols_path ON codeintel_symbols(path);
CREATE INDEX IF NOT EXISTS idx_codeintel_symbols_name ON codeintel_symbols(name);
CREATE TABLE IF NOT EXISTS codeintel_edges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_symbol TEXT NOT NULL,
  to_symbol TEXT NOT NULL,
  edge_type TEXT NOT NULL,
  path TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_codeintel_edges_from ON codeintel_edges(from_symbol);
CREATE INDEX IF NOT EXISTS idx_codeintel_edges_to ON codeintel_edges(to_symbol);
CREATE TABLE IF NOT EXISTS codeintel_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  started_at INTEGER NOT NULL,
  duration_ms INTEGER NOT NULL,
  mode TEXT NOT NULL,
  files_seen INTEGER NOT NULL,
  files_indexed INTEGER NOT NULL,
  files_removed INTEGER NOT NULL,
  symbols INTEGER NOT NULL,
  edges INTEGER NOT NULL,
  status TEXT NOT NULL
);
`)
	if err != nil {
		return fmt.Errorf("codeintel: init schema: %w", err)
	}
	return nil
}

func (s *Store) Index(ctx context.Context, opts IndexOptions) (IndexResult, error) {
	start := time.Now()
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 20000
	}
	files, err := discoverFiles(s.repoRoot, opts.MaxFiles)
	if err != nil {
		return IndexResult{}, err
	}
	current := map[string]fileInfo{}
	for _, f := range files {
		current[f.Path] = f
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return IndexResult{}, fmt.Errorf("codeintel: begin index: %w", err)
	}
	defer tx.Rollback()

	known, err := loadKnownFiles(ctx, tx)
	if err != nil {
		return IndexResult{}, err
	}
	removed := 0
	for path := range known {
		if _, ok := current[path]; ok {
			continue
		}
		if err := deletePath(ctx, tx, path); err != nil {
			return IndexResult{}, err
		}
		removed++
	}
	indexed := 0
	totalSymbols := 0
	totalEdges := 0
	for _, f := range files {
		prev, ok := known[f.Path]
		if !opts.Full && ok && prev.Hash == f.Hash && prev.ModUnix == f.ModUnix && prev.Size == f.Size {
			continue
		}
		ext, err := extractFile(s.repoRoot, f)
		if err != nil {
			continue
		}
		if err := deletePath(ctx, tx, f.Path); err != nil {
			return IndexResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO codeintel_files(path, language, sha256, mtime_unix, size, parser_version, indexed_at, status)
VALUES(?,?,?,?,?,?,?,?)`, f.Path, f.Language, f.Hash, f.ModUnix, f.Size, ParserVersion, time.Now().Unix(), string(FreshnessFresh)); err != nil {
			return IndexResult{}, fmt.Errorf("codeintel: upsert file %s: %w", f.Path, err)
		}
		for _, sym := range ext.Symbols {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO codeintel_symbols(path, language, kind, name, qualified_name, start_line, end_line, signature, search_text)
VALUES(?,?,?,?,?,?,?,?,?)`, sym.Path, sym.Language, sym.Kind, sym.Name, sym.QualifiedName, sym.StartLine, sym.EndLine, sym.Signature, ext.Text+" "+sym.Signature+" "+sym.Name+" "+sym.QualifiedName); err != nil {
				return IndexResult{}, fmt.Errorf("codeintel: insert symbol %s: %w", sym.QualifiedName, err)
			}
		}
		for _, edge := range ext.Edges {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO codeintel_edges(from_symbol, to_symbol, edge_type, path)
VALUES(?,?,?,?)`, edge.From, edge.To, edge.Type, edge.Path); err != nil {
				return IndexResult{}, fmt.Errorf("codeintel: insert edge %s -> %s: %w", edge.From, edge.To, err)
			}
		}
		indexed++
		totalSymbols += len(ext.Symbols)
		totalEdges += len(ext.Edges)
	}
	duration := time.Since(start).Milliseconds()
	status := FreshnessFresh
	if len(files) >= opts.MaxFiles {
		status = FreshnessPartial
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO codeintel_runs(started_at, duration_ms, mode, files_seen, files_indexed, files_removed, symbols, edges, status)
VALUES(?,?,?,?,?,?,?,?,?)`, start.Unix(), duration, indexMode(opts), len(files), indexed, removed, totalSymbols, totalEdges, string(status)); err != nil {
		return IndexResult{}, fmt.Errorf("codeintel: record run: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return IndexResult{}, fmt.Errorf("codeintel: commit index: %w", err)
	}
	msg := "index fresh"
	if status == FreshnessPartial {
		msg = "index partial: file limit reached"
	}
	return IndexResult{Status: status, FilesSeen: len(files), FilesIndexed: indexed, FilesRemoved: removed, Symbols: totalSymbols, Edges: totalEdges, DurationMS: duration, Message: msg}, nil
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	var st Status
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM codeintel_files`).Scan(&st.Files); err != nil {
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM codeintel_symbols`).Scan(&st.Symbols); err != nil {
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM codeintel_edges`).Scan(&st.Edges); err != nil {
		return st, err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT started_at FROM codeintel_runs ORDER BY id DESC LIMIT 1`).Scan(&st.LastRunUnix)
	if st.Files == 0 {
		st.Status = FreshnessMissing
		st.Message = "code-intel index missing; run code_index"
		return st, nil
	}
	stale, err := s.staleFiles(ctx, 0)
	if err != nil {
		return st, err
	}
	newFiles, err := s.newFileCount(ctx, 20_000)
	if err != nil {
		return st, err
	}
	st.StaleFiles = len(stale)
	st.NewFiles = newFiles
	if len(stale) > 0 || newFiles > 0 {
		st.Status = FreshnessStale
		st.Message = "code-intel index has stale files; run code_index"
		return st, nil
	}
	st.Status = FreshnessFresh
	st.Message = "code-intel index fresh"
	return st, nil
}

func (s *Store) Search(ctx context.Context, opts SearchOptions) (SearchResult, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return SearchResult{}, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	terms := strings.Fields(strings.ToLower(opts.Query))
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, qualified_name, kind, language, path, start_line, end_line, signature, search_text
FROM codeintel_symbols
WHERE (? = '' OR kind = ?)
  AND (? = '' OR language = ?)
  AND (? = '' OR path LIKE '%' || ? || '%')
ORDER BY path, start_line`, strings.TrimSpace(opts.Kind), strings.TrimSpace(opts.Kind), strings.TrimSpace(opts.Language), strings.TrimSpace(opts.Language), strings.TrimSpace(opts.Path), strings.TrimSpace(opts.Path))
	if err != nil {
		return SearchResult{}, fmt.Errorf("codeintel: search: %w", err)
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var sym Symbol
		var text string
		if err := rows.Scan(&sym.ID, &sym.Name, &sym.QualifiedName, &sym.Kind, &sym.Language, &sym.Path, &sym.StartLine, &sym.EndLine, &sym.Signature, &text); err != nil {
			return SearchResult{}, err
		}
		score := scoreSymbol(sym, text, terms)
		if len(terms) > 0 && score == 0 {
			continue
		}
		sym.Score = score
		sym.Freshness = string(status.Status)
		out = append(out, sym)
	}
	if err := rows.Err(); err != nil {
		return SearchResult{}, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].QualifiedName < out[j].QualifiedName
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return SearchResult{Status: status.Status, Results: out, Message: status.Message}, nil
}

func (s *Store) Snippet(ctx context.Context, q string) (SnippetResult, error) {
	sym, status, err := s.findSymbol(ctx, q)
	if err != nil {
		return SnippetResult{}, err
	}
	source, err := readLinesContained(s.repoRoot, sym.Path, sym.StartLine, sym.EndLine)
	if err != nil {
		return SnippetResult{}, err
	}
	return SnippetResult{Status: status.Status, Symbol: sym, Source: source, Message: status.Message}, nil
}

func (s *Store) Trace(ctx context.Context, q, direction string) (TraceResult, error) {
	sym, status, err := s.findSymbol(ctx, q)
	if err != nil {
		return TraceResult{}, err
	}
	if direction == "" {
		direction = "both"
	}
	var rows *sql.Rows
	switch direction {
	case "inbound":
		rows, err = s.db.QueryContext(ctx, `SELECT from_symbol, to_symbol, edge_type, path FROM codeintel_edges WHERE to_symbol = ? ORDER BY from_symbol`, sym.QualifiedName)
	case "outbound":
		rows, err = s.db.QueryContext(ctx, `SELECT from_symbol, to_symbol, edge_type, path FROM codeintel_edges WHERE from_symbol = ? ORDER BY to_symbol`, sym.QualifiedName)
	default:
		rows, err = s.db.QueryContext(ctx, `SELECT from_symbol, to_symbol, edge_type, path FROM codeintel_edges WHERE from_symbol = ? OR to_symbol = ? ORDER BY edge_type, from_symbol, to_symbol`, sym.QualifiedName, sym.QualifiedName)
	}
	if err != nil {
		return TraceResult{}, err
	}
	defer rows.Close()
	var edges []TraceEdge
	for rows.Next() {
		var e TraceEdge
		if err := rows.Scan(&e.From, &e.To, &e.Type, &e.Path); err != nil {
			return TraceResult{}, err
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return TraceResult{}, err
	}
	return TraceResult{Status: status.Status, Symbol: sym, Edges: edges, Message: status.Message}, nil
}

func (s *Store) Impact(ctx context.Context, paths []string, baseRef string) (ImpactResult, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return ImpactResult{}, err
	}
	if len(paths) == 0 {
		paths, err = changedPaths(ctx, s.repoRoot, baseRef)
		if err != nil {
			return ImpactResult{}, err
		}
	}
	paths = cleanRelPaths(paths)
	if len(paths) == 0 {
		return ImpactResult{Status: status.Status, ChangedPaths: []string{}, Symbols: []Symbol{}, Tests: []string{}, Docs: []string{}, Features: []string{}, Tickets: []string{}, Message: status.Message}, nil
	}
	syms, err := s.symbolsForPaths(ctx, paths)
	if err != nil {
		return ImpactResult{}, err
	}
	tests := likelyTests(s.repoRoot, paths, syms)
	docs, features, tickets := relatedDocs(s.repoRoot, paths, syms)
	return ImpactResult{Status: status.Status, ChangedPaths: paths, Symbols: syms, Tests: tests, Docs: docs, Features: features, Tickets: tickets, Message: status.Message}, nil
}

func (s *Store) findSymbol(ctx context.Context, q string) (Symbol, Status, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return Symbol{}, status, err
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return Symbol{}, status, fmt.Errorf("codeintel: symbol query is required")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, qualified_name, kind, language, path, start_line, end_line, signature
FROM codeintel_symbols
WHERE qualified_name = ? OR name = ?
ORDER BY CASE WHEN qualified_name = ? THEN 0 ELSE 1 END, length(qualified_name)
LIMIT 1`, q, q, q)
	var sym Symbol
	if err := row.Scan(&sym.ID, &sym.Name, &sym.QualifiedName, &sym.Kind, &sym.Language, &sym.Path, &sym.StartLine, &sym.EndLine, &sym.Signature); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Symbol{}, status, fmt.Errorf("codeintel: symbol %q not found; run code_search first", q)
		}
		return Symbol{}, status, err
	}
	sym.Freshness = string(status.Status)
	return sym, status, nil
}

func (s *Store) symbolsForPaths(ctx context.Context, paths []string) ([]Symbol, error) {
	seen := map[string]bool{}
	var out []Symbol
	for _, p := range paths {
		rows, err := s.db.QueryContext(ctx, `
SELECT id, name, qualified_name, kind, language, path, start_line, end_line, signature
FROM codeintel_symbols WHERE path = ? ORDER BY start_line`, p)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var sym Symbol
			if err := rows.Scan(&sym.ID, &sym.Name, &sym.QualifiedName, &sym.Kind, &sym.Language, &sym.Path, &sym.StartLine, &sym.EndLine, &sym.Signature); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if !seen[sym.QualifiedName] {
				out = append(out, sym)
				seen[sym.QualifiedName] = true
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return out, nil
}

func loadKnownFiles(ctx context.Context, tx *sql.Tx) (map[string]fileInfo, error) {
	rows, err := tx.QueryContext(ctx, `SELECT path, language, sha256, mtime_unix, size FROM codeintel_files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]fileInfo{}
	for rows.Next() {
		var f fileInfo
		if err := rows.Scan(&f.Path, &f.Language, &f.Hash, &f.ModUnix, &f.Size); err != nil {
			return nil, err
		}
		out[f.Path] = f
	}
	return out, nil
}

func deletePath(ctx context.Context, tx *sql.Tx, path string) error {
	for _, q := range []string{
		`DELETE FROM codeintel_files WHERE path = ?`,
		`DELETE FROM codeintel_symbols WHERE path = ?`,
		`DELETE FROM codeintel_edges WHERE path = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, path); err != nil {
			return err
		}
	}
	return nil
}

func indexMode(opts IndexOptions) string {
	if opts.Full {
		return "full"
	}
	return "incremental"
}

func discoverFiles(root string, max int) ([]fileInfo, error) {
	var out []fileInfo
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if skipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(out) >= max {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 || skipFile(name) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		lang := languageFor(rel)
		if lang == "" {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 2<<20 {
			return nil
		}
		hash, err := hashFile(path)
		if err != nil {
			return nil
		}
		out = append(out, fileInfo{Path: rel, Language: lang, Hash: hash, ModUnix: info.ModTime().Unix(), Size: info.Size()})
		return nil
	})
	return out, err
}

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "dist", "build", ".next", ".cache", "coverage", ".mars-harness":
		return true
	default:
		return false
	}
}

func skipFile(name string) bool {
	return strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".gif") || strings.HasSuffix(name, ".pdf") || strings.HasSuffix(name, ".zip")
}

func languageFor(path string) string {
	base := filepath.Base(path)
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return "typescript"
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"), strings.HasSuffix(path, ".mjs"), strings.HasSuffix(path, ".cjs"):
		return "javascript"
	case strings.HasSuffix(path, ".md"):
		return "markdown"
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "yaml"
	case strings.HasSuffix(path, ".json") || base == "package.json":
		return "json"
	default:
		return ""
	}
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func extractFile(root string, f fileInfo) (extracted, error) {
	abs := filepath.Join(root, filepath.FromSlash(f.Path))
	data, err := os.ReadFile(abs)
	if err != nil {
		return extracted{}, err
	}
	text := string(data)
	switch f.Language {
	case "go":
		return extractGo(f.Path, text)
	case "javascript", "typescript":
		return extractJSLike(f.Path, f.Language, text), nil
	case "markdown":
		return extractMarkdown(f.Path, text), nil
	case "json", "yaml":
		return extractConfig(f.Path, f.Language, text), nil
	default:
		return extracted{Text: text}, nil
	}
}

func extractGo(path, text string) (extracted, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, text, parser.ParseComments)
	if err != nil {
		return extracted{Text: text}, nil
	}
	var out extracted
	out.Text = text
	pkg := file.Name.Name
	imports := map[string]string{}
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		name := filepath.Base(importPath)
		if imp.Name != nil && imp.Name.Name != "." && imp.Name.Name != "_" {
			name = imp.Name.Name
		}
		imports[name] = importPath
		out.Edges = append(out.Edges, TraceEdge{From: pkg, To: importPath, Type: "imports", Path: path})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		name := fn.Name.Name
		kind := "function"
		receiver := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			kind = "method"
			receiver = exprString(fn.Recv.List[0].Type)
		}
		qn := pkg + "." + name
		if receiver != "" {
			qn = pkg + "." + strings.Trim(receiver, "*") + "." + name
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.End()).Line
		out.Symbols = append(out.Symbols, Symbol{Name: name, QualifiedName: qn, Kind: kind, Language: "go", Path: path, StartLine: start, EndLine: end, Signature: goSignature(fn)})
		ast.Inspect(fn.Body, func(child ast.Node) bool {
			call, ok := child.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee := callName(call.Fun, imports)
			if callee != "" {
				out.Edges = append(out.Edges, TraceEdge{From: qn, To: callee, Type: "calls", Path: path})
			}
			return true
		})
		return false
	})
	return out, nil
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	default:
		return ""
	}
}

func callName(e ast.Expr, imports map[string]string) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		left := exprString(v.X)
		if imp, ok := imports[left]; ok {
			return imp + "." + v.Sel.Name
		}
		return left + "." + v.Sel.Name
	default:
		return ""
	}
}

func goSignature(fn *ast.FuncDecl) string {
	recv := ""
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv = "(" + exprString(fn.Recv.List[0].Type) + ") "
	}
	return "func " + recv + fn.Name.Name
}

var (
	jsFuncRe   = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(`)
	jsClassRe  = regexp.MustCompile(`(?m)^\s*(?:export\s+)?class\s+([A-Za-z_$][\w$]*)`)
	jsConstRe  = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>`)
	jsImportRe = regexp.MustCompile(`(?m)^\s*import\s+.*?\s+from\s+['"]([^'"]+)['"]`)
	mdHeadRe   = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*$`)
)

func extractJSLike(path, lang, text string) extracted {
	var out extracted
	out.Text = text
	for _, m := range jsImportRe.FindAllStringSubmatch(text, -1) {
		out.Edges = append(out.Edges, TraceEdge{From: path, To: m[1], Type: "imports", Path: path})
	}
	addRegexSymbols(&out, path, lang, "function", text, jsFuncRe)
	addRegexSymbols(&out, path, lang, "class", text, jsClassRe)
	addRegexSymbols(&out, path, lang, "function", text, jsConstRe)
	if filepath.Base(path) == "package.json" {
		out.Symbols = append(out.Symbols, Symbol{Name: "package", QualifiedName: path + "#package", Kind: "config", Language: lang, Path: path, StartLine: 1, EndLine: lineCount(text), Signature: "package.json"})
	}
	return out
}

func extractMarkdown(path, text string) extracted {
	var out extracted
	out.Text = text
	for _, m := range mdHeadRe.FindAllStringSubmatchIndex(text, -1) {
		name := strings.TrimSpace(text[m[4]:m[5]])
		line := 1 + strings.Count(text[:m[0]], "\n")
		out.Symbols = append(out.Symbols, Symbol{Name: name, QualifiedName: path + "#" + slug(name), Kind: "heading", Language: "markdown", Path: path, StartLine: line, EndLine: line, Signature: name})
	}
	return out
}

func extractConfig(path, lang, text string) extracted {
	name := filepath.Base(path)
	return extracted{Text: text, Symbols: []Symbol{{Name: name, QualifiedName: path, Kind: "config", Language: lang, Path: path, StartLine: 1, EndLine: lineCount(text), Signature: name}}}
}

func addRegexSymbols(out *extracted, path, lang, kind, text string, re *regexp.Regexp) {
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		qn := strings.TrimSuffix(path, filepath.Ext(path)) + "." + name
		out.Symbols = append(out.Symbols, Symbol{Name: name, QualifiedName: filepath.ToSlash(qn), Kind: kind, Language: lang, Path: path, StartLine: line, EndLine: line, Signature: kind + " " + name})
	}
}

func lineCount(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func scoreSymbol(sym Symbol, text string, terms []string) int {
	if len(terms) == 0 {
		return 1
	}
	hay := strings.ToLower(sym.Name + " " + sym.QualifiedName + " " + sym.Kind + " " + sym.Path + " " + text)
	score := 0
	for _, term := range terms {
		if strings.Contains(strings.ToLower(sym.Name), term) {
			score += 20
		}
		if strings.Contains(strings.ToLower(sym.QualifiedName), term) {
			score += 10
		}
		if strings.Contains(hay, term) {
			score += 5
		}
	}
	return score
}

func (s *Store) staleFiles(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, sha256, mtime_unix, size FROM codeintel_files ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var path, oldHash string
		var mtime, size int64
		if err := rows.Scan(&path, &oldHash, &mtime, &size); err != nil {
			return nil, err
		}
		abs := filepath.Join(s.repoRoot, filepath.FromSlash(path))
		info, err := os.Stat(abs)
		if err != nil {
			stale = append(stale, path)
		} else if info.ModTime().Unix() != mtime || info.Size() != size {
			hash, err := hashFile(abs)
			if err != nil || hash != oldHash {
				stale = append(stale, path)
			}
		}
		if limit > 0 && len(stale) >= limit {
			break
		}
	}
	return stale, rows.Err()
}

func (s *Store) newFileCount(ctx context.Context, maxFiles int) (int, error) {
	files, err := discoverFiles(s.repoRoot, maxFiles)
	if err != nil {
		return 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT path FROM codeintel_files`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	known := map[string]bool{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return 0, err
		}
		known[path] = true
	}
	count := 0
	for _, f := range files {
		if !known[f.Path] {
			count++
		}
	}
	return count, rows.Err()
}

func readLinesContained(root, rel string, start, end int) (string, error) {
	clean, err := containedPath(root, rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if start < 1 {
		start = 1
	}
	if end < start || end > len(lines) {
		end = len(lines)
	}
	if start > len(lines) {
		return "", fmt.Errorf("codeintel: line %d beyond %s", start, rel)
	}
	return strings.Join(lines[start-1:end], "\n"), nil
}

func containedPath(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("codeintel: path must be relative")
	}
	clean := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	got, err := filepath.Rel(root, clean)
	if err != nil || got == ".." || strings.HasPrefix(got, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("codeintel: path escapes repo root")
	}
	return clean, nil
}

func changedPaths(ctx context.Context, root, baseRef string) ([]string, error) {
	var cmd *exec.Cmd
	if strings.TrimSpace(baseRef) != "" {
		cmd = exec.CommandContext(ctx, "git", "diff", "--name-only", baseRef+"...HEAD")
	} else {
		cmd = exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "-uall")
	}
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	var paths []string
	for _, rawLine := range strings.Split(string(out), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if baseRef == "" {
			if len(line) > 3 {
				line = line[3:]
			}
			if idx := strings.LastIndex(line, " -> "); idx >= 0 {
				line = line[idx+4:]
			}
		}
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, filepath.ToSlash(line))
		}
	}
	return cleanRelPaths(paths), nil
}

func cleanRelPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		p = filepath.ToSlash(strings.TrimSpace(p))
		p = strings.TrimPrefix(p, "./")
		if p == "" || strings.HasPrefix(p, "../") || filepath.IsAbs(p) || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func likelyTests(root string, paths []string, syms []Symbol) []string {
	candidates := map[string]bool{}
	for _, p := range paths {
		ext := filepath.Ext(p)
		base := strings.TrimSuffix(p, ext)
		switch ext {
		case ".go":
			candidates[base+"_test.go"] = true
		case ".ts", ".tsx", ".js", ".jsx":
			candidates[base+".test"+ext] = true
			candidates[base+".spec"+ext] = true
		}
	}
	for _, sym := range syms {
		if sym.Language == "go" {
			candidates[strings.TrimSuffix(sym.Path, ".go")+"_test.go"] = true
		}
	}
	var out []string
	for p := range candidates {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err == nil {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func relatedDocs(root string, paths []string, syms []Symbol) ([]string, []string, []string) {
	needles := map[string]bool{}
	for _, p := range paths {
		needles[p] = true
		needles[filepath.Base(p)] = true
	}
	for _, sym := range syms {
		needles[sym.Name] = true
		needles[sym.QualifiedName] = true
	}
	for needle := range needles {
		if strings.TrimSpace(needle) == "" {
			delete(needles, needle)
		}
	}
	if len(needles) == 0 {
		return []string{}, []string{}, []string{}
	}
	docs := map[string]bool{}
	features := map[string]bool{}
	tickets := map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(root, "docs"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		match := false
		for needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		switch {
		case strings.HasPrefix(rel, "docs/features/"):
			features[rel] = true
		case strings.HasPrefix(rel, "docs/tickets/"):
			tickets[rel] = true
		default:
			docs[rel] = true
		}
		return nil
	})
	return sortedKeys(docs), sortedKeys(features), sortedKeys(tickets)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

func Marshal(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}
