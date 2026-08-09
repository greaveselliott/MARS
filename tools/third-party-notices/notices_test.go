package notices

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	golicense "github.com/google/go-licenses/v2/licenses"
)

func TestRepositoryPolicyInputsAreExact(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	p, err := LoadPolicy(repo)
	if err != nil {
		t.Fatalf("load repository policy: %v", err)
	}
	if p.GoLicenses.Version != "v2.0.1" || p.Toolchain.Version != "go1.26.5" || len(p.BrowserAssets) != 2 || len(p.Overrides) != 3 {
		t.Fatalf("unexpected exact policy: %+v", p)
	}
}

func TestRenderThirdPartyNoticesUsesCanonicalBrowserLicenses(t *testing.T) {
	p := expectedPolicy
	p.inputs = map[string]string{
		p.BrowserAssets[0].Path: "canonical htmx license\n",
		p.BrowserAssets[1].Path: "canonical chart license\n",
		p.Toolchain.LicensePath: "tool license\n",
		p.Toolchain.PatentsPath: "tool patents\n",
	}
	row := Dependency{
		Library: "example.test/pkg", Module: "example.test/mod", Version: "v1.0.0",
		Packages: []string{"example.test/pkg"}, License: "Apache-2.0 AND BSD-3-Clause", LicenseText: "combined license\n",
	}
	got, err := RenderThirdPartyNotices(p, []Dependency{row})
	if err != nil {
		t.Fatalf("RenderThirdPartyNotices: %v", err)
	}
	for _, want := range []string{"htmx 2.0.4", "canonical htmx license", "Chart.js 4.4.7", "canonical chart license", "combined license"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("output missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(string(got)), "provisional") {
		t.Fatal("output retained provisional wording")
	}
}

func TestRequireOverridesUsedRejectsMissingExtraAndStale(t *testing.T) {
	p := expectedPolicy
	all := map[string]bool{}
	for _, item := range p.Overrides {
		all[item.Module+"@"+item.Version] = true
	}
	if err := requireOverridesUsed(p, all); err != nil {
		t.Fatalf("exact overrides rejected: %v", err)
	}
	missing := map[string]bool{}
	for key := range all {
		missing[key] = true
		break
	}
	if err := requireOverridesUsed(p, missing); err == nil {
		t.Fatal("missing overrides accepted")
	}
	extra := map[string]bool{}
	for key := range all {
		extra[key] = true
	}
	extra["example.test/unreviewed@v1.0.0"] = true
	if err := requireOverridesUsed(p, extra); err == nil {
		t.Fatal("extra override use accepted")
	}
}

func TestLoadPolicyRejectsAmbiguousOrExtendedJSON(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	base, err := os.ReadFile(filepath.Join(repo, policyPath))
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string][]byte{
		"duplicate key":  []byte(strings.Replace(string(base), "{\n  \"schema_version\": 1,", "{\n  \"schema_version\": 1,\n  \"schema_version\": 1,", 1)),
		"unknown key":    []byte(strings.Replace(string(base), "{\n  \"schema_version\": 1,", "{\n  \"schema_version\": 1,\n  \"unexpected\": true,", 1)),
		"trailing value": append(slices.Clone(base), []byte("{}\n")...),
	}
	for name, data := range fixtures {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, policyPath)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadPolicy(root); err == nil || !strings.Contains(err.Error(), "SHA-256") {
				t.Fatalf("LoadPolicy error = %v", err)
			}
		})
	}
}

func TestGoLicensesBuildDependencyIsExactlyPinned(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	for _, item := range []struct {
		name string
		want string
	}{
		{"go.mod", "\t" + expectedPolicy.GoLicenses.Module + " " + expectedPolicy.GoLicenses.Version + "\n"},
		{"go.sum", expectedPolicy.GoLicenses.Module + " " + expectedPolicy.GoLicenses.Version + " " + expectedPolicy.GoLicenses.ModuleSum + "\n"},
	} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(file), item.name))
		if err != nil {
			t.Fatalf("read %s: %v", item.name, err)
		}
		if strings.Count(string(data), item.want) != 1 {
			t.Fatalf("%s does not contain exactly one reviewed pin %q", item.name, item.want)
		}
	}
}

func TestValidateExactPolicyRejectsDrift(t *testing.T) {
	tests := map[string]func(*Policy){
		"tool version": func(p *Policy) { p.GoLicenses.Version = "v2.0.2" },
		"allowed list": func(p *Policy) { p.AllowedGoLicenses = append(p.AllowedGoLicenses, "GPL-3.0") },
		"override":     func(p *Policy) { p.Overrides[0].Version = "v0.0.0-unreviewed" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			p := expectedPolicy
			p.AllowedGoLicenses = slices.Clone(expectedPolicy.AllowedGoLicenses)
			p.Overrides = slices.Clone(expectedPolicy.Overrides)
			mutate(&p)
			if err := validateExactPolicy(p); err == nil {
				t.Fatal("drift accepted")
			}
		})
	}
}

func TestDependencyFromLibrarySortsIdentity(t *testing.T) {
	lib := &golicense.Library{
		Packages: []string{"example.test/z", "example.test/a", "example.test/a"},
		Licenses: []golicense.License{{Name: "MIT"}},
	}
	got, err := DependencyFromLibrary(lib, "example.test/mod", "v1.2.3", "license\n", nil)
	if err != nil {
		t.Fatalf("convert library: %v", err)
	}
	if !slices.Equal(got.Packages, []string{"example.test/a", "example.test/z"}) {
		t.Fatalf("packages = %v", got.Packages)
	}
}

func TestRenderIsDeterministicAndComplete(t *testing.T) {
	p := expectedPolicy
	p.inputs = map[string]string{
		p.Toolchain.LicensePath: "tool license\n",
		p.Toolchain.PatentsPath: "tool patents\n",
	}
	alpha := Dependency{
		Library: "example.test/alpha", Module: "example.test/alpha", Version: "v1.0.0",
		Packages: []string{"example.test/alpha"}, License: "MIT", LicenseText: "alpha license\n",
		Notices: []Text{{Name: "SECOND", Text: "second notice\n"}, {Name: "NOTICE", Text: "alpha notice\n"}},
	}
	beta := Dependency{
		Library: "example.test/beta", Module: "example.test/beta", Version: "v2.0.0",
		Packages: []string{"example.test/beta"}, License: "BSD-3-Clause", LicenseText: "beta license\r\n",
	}
	first, err := Render(p, []Dependency{beta, alpha})
	if err != nil {
		t.Fatalf("render first: %v", err)
	}
	second, err := Render(p, []Dependency{alpha, beta})
	if err != nil {
		t.Fatalf("render second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("render differs when input ordering changes")
	}
	alpha.Notices[0], alpha.Notices[1] = alpha.Notices[1], alpha.Notices[0]
	third, err := Render(p, []Dependency{alpha, beta})
	if err != nil {
		t.Fatalf("render reversed notices: %v", err)
	}
	if string(first) != string(third) {
		t.Fatal("render differs when NOTICE ordering changes")
	}
	text := string(first)
	for _, want := range []string{"alpha license", "alpha notice", "beta license", "tool license", "tool patents", "go-licenses/v2@v2.0.1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("render missing %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "example.test/alpha@") > strings.Index(text, "example.test/beta@") {
		t.Fatal("dependency sections are not sorted")
	}
}

func TestRenderRejectsUnreviewedLicensesAndDuplicateRows(t *testing.T) {
	p := expectedPolicy
	p.inputs = map[string]string{
		p.Toolchain.LicensePath: "tool license\n",
		p.Toolchain.PatentsPath: "tool patents\n",
	}
	base := Dependency{
		Library: "example.test/pkg", Module: "example.test/mod", Version: "v1.0.0",
		Packages: []string{"example.test/pkg"}, License: "MIT", LicenseText: "license\n",
	}
	for _, license := range []string{"Unknown", "GPL-3.0", "AGPL-3.0", "SSPL-1.0"} {
		t.Run(license, func(t *testing.T) {
			item := base
			item.License = license
			if _, err := Render(p, []Dependency{item}); err == nil || !strings.Contains(err.Error(), "unreviewed license") {
				t.Fatalf("Render license %q error = %v", license, err)
			}
		})
	}
	if _, err := Render(p, []Dependency{base, base}); err == nil || !strings.Contains(err.Error(), "duplicate dependency") {
		t.Fatalf("duplicate error = %v", err)
	}
	for name, notices := range map[string][]Text{
		"duplicate":          {{Name: "NOTICE", Text: "first\n"}, {Name: "NOTICE", Text: "second\n"}},
		"empty":              {{Name: "NOTICE", Text: " \n"}},
		"oversized text":     {{Name: "NOTICE", Text: strings.Repeat("x", maxInputSize+1)}},
		"invalid UTF-8 name": {{Name: string([]byte{0xff}), Text: "notice\n"}},
		"oversized name":     {{Name: strings.Repeat("n", maxInputSize+1), Text: "notice\n"}},
	} {
		t.Run("notice "+name, func(t *testing.T) {
			item := base
			item.Notices = notices
			if _, err := Render(p, []Dependency{item}); err == nil {
				t.Fatal("invalid NOTICE accepted")
			}
		})
	}
}

func TestRenderPreservesDistinctLicenseRowsWithSameLibraryName(t *testing.T) {
	p := expectedPolicy
	p.inputs = map[string]string{
		p.Toolchain.LicensePath: "tool license\n",
		p.Toolchain.PatentsPath: "tool patents\n",
	}
	first := Dependency{
		Library: "example.test/shared", Module: "example.test/mod", Version: "v1.0.0", Identity: "LICENSE",
		Packages: []string{"example.test/shared/a"}, License: "MIT", LicenseText: "first license\n",
	}
	second := Dependency{
		Library: "example.test/shared", Module: "example.test/mod", Version: "v1.0.0", Identity: "sub/LICENSE",
		Packages: []string{"example.test/shared/b"}, License: "MIT", LicenseText: "second license\n",
	}
	got, err := Render(p, []Dependency{first, second})
	if err != nil {
		t.Fatalf("Render distinct rows: %v", err)
	}
	if strings.Count(string(got), "example.test/mod@v1.0.0 — MIT") != 2 {
		t.Fatalf("distinct license identities collapsed:\n%s", got)
	}
}

func TestReadBoundedTextRejectsSymlinkNonUTF8AndDigestDrift(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input"), []byte("reviewed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedText(dir, "input", strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("digest error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "non-utf8"), []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedText(dir, "non-utf8", ""); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("UTF-8 error = %v", err)
	}
	if err := os.Symlink("input", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedText(dir, "link", ""); err == nil || !strings.Contains(err.Error(), "without following links") {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestReadBoundedTextRejectsParentSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "input"), []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "parent")); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedText(root, "parent/input", ""); err == nil {
		t.Fatal("symlinked parent accepted")
	}
}

func TestOpenedInputIsBoundToDescriptorAndSizeLimited(t *testing.T) {
	root := t.TempDir()
	original := []byte("reviewed\n")
	path := filepath.Join(root, "input")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := openRepositoryFileNoFollow(root, "input")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := os.Rename(path, filepath.Join(root, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(original)
	got, err := readOpenedBoundedText(file, "input", hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatalf("read descriptor-bound input: %v", err)
	}
	if got != string(original) {
		t.Fatalf("read replacement: %q", got)
	}

	growingPath := filepath.Join(root, "growing")
	if err := os.WriteFile(growingPath, []byte("small\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	growing, err := openRepositoryFileNoFollow(root, "growing")
	if err != nil {
		t.Fatal(err)
	}
	defer growing.Close()
	if err := os.WriteFile(growingPath, []byte(strings.Repeat("x", maxInputSize+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readOpenedBoundedText(growing, "growing", ""); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("growth error = %v", err)
	}
}
