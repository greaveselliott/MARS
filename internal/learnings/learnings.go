package learnings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const fileName = "learnings.yaml"

// Learnings is the per-repo knowledge store persisted at .harness/learnings.yaml.
type Learnings struct {
	Conventions Conventions `yaml:"conventions"`
	Lessons     []Lesson    `yaml:"lessons,omitempty"`
	Excludes    []string    `yaml:"excludes,omitempty"`
}

// Conventions holds detected repo-level conventions.
type Conventions struct {
	PackageManager string `yaml:"package_manager,omitempty"`
	TestCommand    string `yaml:"test_command,omitempty"`
	LintCommand    string `yaml:"lint_command,omitempty"`
	BuildCommand   string `yaml:"build_command,omitempty"`
	Framework      string `yaml:"framework,omitempty"`
	Language       string `yaml:"language,omitempty"`
}

// Lesson records a single piece of knowledge extracted from agent runs.
type Lesson struct {
	ID      string `yaml:"id"`
	Role    string `yaml:"role"`
	Created string `yaml:"created"`
	Type    string `yaml:"type"`
	Content string `yaml:"content"`
}

// Store manages reading and writing the per-repo learnings file.
type Store struct {
	mu       sync.Mutex
	repoRoot string
}

// NewStore creates a store rooted at the given repo path.
func NewStore(repoRoot string) *Store {
	return &Store{repoRoot: repoRoot}
}

func (s *Store) path() string {
	return filepath.Join(s.repoRoot, ".harness", fileName)
}

// Load reads the learnings file, returning an empty Learnings if it doesn't exist.
func (s *Store) Load() (*Learnings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnsafe()
}

func (s *Store) loadUnsafe() (*Learnings, error) {
	data, err := os.ReadFile(s.path())
	if os.IsNotExist(err) {
		return &Learnings{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("learnings: read %s: %w", s.path(), err)
	}
	var l Learnings
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("learnings: parse %s: %w", s.path(), err)
	}
	return &l, nil
}

// Save writes the learnings file to .harness/learnings.yaml.
func (s *Store) Save(l *Learnings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnsafe(l)
}

func (s *Store) saveUnsafe(l *Learnings) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return fmt.Errorf("learnings: marshal: %w", err)
	}
	dir := filepath.Dir(s.path())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("learnings: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(s.path(), data, 0o644); err != nil {
		return fmt.Errorf("learnings: write %s: %w", s.path(), err)
	}
	return nil
}

// SetConventions updates the conventions section and saves.
func (s *Store) SetConventions(conv Conventions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	l.Conventions = conv
	return s.saveUnsafe(l)
}

// AddLesson appends a lesson if no duplicate content exists. Returns true if added.
func (s *Store) AddLesson(role, lessonType, content string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, err := s.loadUnsafe()
	if err != nil {
		return false, err
	}

	for _, existing := range l.Lessons {
		if existing.Content == content {
			return false, nil
		}
	}

	nextID := fmt.Sprintf("lesson-%03d", len(l.Lessons)+1)
	l.Lessons = append(l.Lessons, Lesson{
		ID:      nextID,
		Role:    role,
		Created: time.Now().UTC().Format("2006-01-02"),
		Type:    lessonType,
		Content: content,
	})

	if err := s.saveUnsafe(l); err != nil {
		return false, err
	}
	return true, nil
}

// FormatForContext renders the learnings as a text block suitable for
// injection into the agent's system prompt.
func (l *Learnings) FormatForContext() string {
	if l == nil {
		return ""
	}
	var b strings.Builder

	c := l.Conventions
	if c.PackageManager != "" || c.Framework != "" || c.Language != "" {
		b.WriteString("### Conventions\n")
		if c.PackageManager != "" {
			fmt.Fprintf(&b, "- Package manager: %s\n", c.PackageManager)
		}
		if c.Language != "" {
			fmt.Fprintf(&b, "- Language: %s\n", c.Language)
		}
		if c.Framework != "" {
			fmt.Fprintf(&b, "- Framework: %s\n", c.Framework)
		}
		if c.TestCommand != "" {
			fmt.Fprintf(&b, "- Test command: %s\n", c.TestCommand)
		}
		if c.LintCommand != "" {
			fmt.Fprintf(&b, "- Lint command: %s\n", c.LintCommand)
		}
		if c.BuildCommand != "" {
			fmt.Fprintf(&b, "- Build command: %s\n", c.BuildCommand)
		}
		b.WriteString("\n")
	}

	if len(l.Lessons) > 0 {
		b.WriteString("### Lessons from past runs\n")
		for _, lesson := range l.Lessons {
			fmt.Fprintf(&b, "- (%s) %s\n", lesson.Role, lesson.Content)
		}
		b.WriteString("\n")
	}

	if len(l.Excludes) > 0 {
		b.WriteString("### Excluded directories\n")
		fmt.Fprintf(&b, "- %s\n", strings.Join(l.Excludes, ", "))
	}

	return strings.TrimSpace(b.String())
}

// SetExcludes updates the excluded directories list.
func (s *Store) SetExcludes(excludes []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, err := s.loadUnsafe()
	if err != nil {
		return err
	}
	l.Excludes = excludes
	return s.saveUnsafe(l)
}
