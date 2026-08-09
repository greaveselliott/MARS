// MarsDocSync:
// docs:
// - docs/features/F-017-open-source-publication.md
// - docs/features/F-018-goreleaser-distribution.md

// Package notices contains the build-only deterministic dependency-notice
// policy. It is intentionally a separate Go module so the pinned inspection
// tool never becomes a dependency of the MARS runtime.
package notices

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	golicense "github.com/google/go-licenses/v2/licenses"
)

const (
	policyPath   = "third_party/licenses/overrides.json"
	policySHA256 = "d97749ffdaa7525cf4455e4e6562addfc8544e0cd675edf05556e4e418984b21"
	maxInputSize = 1 << 20
)

var expectedPolicy = Policy{
	SchemaVersion: 1,
	GoLicenses: ToolPin{
		Module:       "github.com/google/go-licenses/v2",
		Version:      "v2.0.1",
		SourceCommit: "3e084b0caf710f7bfead967567539214f598c0a2",
		ModuleSum:    "h1:ti+9bi5o7DKbeeg5eBb/uZTgsaPNoJaLCh93cRcXsW8=",
	},
	AllowedGoLicenses: []string{"Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "MIT"},
	Toolchain: ToolchainInput{
		Version:       "go1.26.5",
		LicensePath:   "third_party/licenses/go-1.26.5/LICENSE",
		LicenseSHA256: "911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad",
		PatentsPath:   "third_party/licenses/go-1.26.5/PATENTS",
		PatentsSHA256: "96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc",
	},
	Overrides: []Override{
		{
			Module: "github.com/cyberphone/json-canonicalization", Version: "v0.0.0-20241213102144-19d51d7fe467",
			SourceCommit: "19d51d7fe467d4706a3ff08adf8a748f29fc21e0", ModuleSum: "h1:uX1JmpONuD549D73r6cgnxyUu18Zb7yHAy5AYU0Pm4Q=",
			License: "Apache-2.0", LicensePath: "third_party/licenses/json-canonicalization/LICENSE",
			LicenseSHA256: "6821faaddedf2d78c95bb6d98b127e9e616097afd2f6bcc34389f000d13ab12d",
		},
		{
			Module: "github.com/in-toto/attestation", Version: "v1.2.0",
			SourceCommit: "df02077bf97218a8860a5c534eff1f1381f56984", ModuleSum: "h1:aPRUZ3azbqD7yEBD5fP3TD8Dszf+YHo284SOcpahjQk=",
			License: "Apache-2.0", LicensePath: "third_party/licenses/in-toto-attestation/LICENSE",
			LicenseSHA256: "b3a6ac899861c2c28a1abd5c7ea8733fefaed6938d730e099c42671386032aeb",
		},
		{
			Module: "github.com/in-toto/in-toto-golang", Version: "v0.11.0",
			SourceCommit: "36d782ffb2ca3adbffcdce1fd971c23319dd4469", ModuleSum: "h1:nfidMYBFx+E0lnmX5KUnN2Pdm8zdNKal1ayjJuzzRoA=",
			License: "Apache-2.0", LicensePath: "third_party/licenses/in-toto-golang/LICENSE",
			LicenseSHA256: "b5c26c8a2ad6dd7cac6646e470a32ee42d23c36124caac4e66c989b6545b2a44",
		},
	},
}

// Policy is the immutable build-time dependency-notice policy.
type Policy struct {
	SchemaVersion     int            `json:"schema_version"`
	GoLicenses        ToolPin        `json:"go_licenses"`
	AllowedGoLicenses []string       `json:"allowed_go_licenses"`
	Toolchain         ToolchainInput `json:"toolchain"`
	Overrides         []Override     `json:"overrides"`

	inputs map[string]string
}

type ToolPin struct {
	Module       string `json:"module"`
	Version      string `json:"version"`
	SourceCommit string `json:"source_commit"`
	ModuleSum    string `json:"module_sum"`
}

type ToolchainInput struct {
	Version       string `json:"version"`
	LicensePath   string `json:"license_path"`
	LicenseSHA256 string `json:"license_sha256"`
	PatentsPath   string `json:"patents_path"`
	PatentsSHA256 string `json:"patents_sha256"`
}

type Override struct {
	Module        string `json:"module"`
	Version       string `json:"version"`
	SourceCommit  string `json:"source_commit"`
	ModuleSum     string `json:"module_sum"`
	License       string `json:"license"`
	LicensePath   string `json:"license_path"`
	LicenseSHA256 string `json:"license_sha256"`
}

// Dependency is one license-bearing go-licenses library row. Module and
// version are explicit so later four-platform collection can reconcile the
// library rows against the independent go list module inventory.
type Dependency struct {
	Library     string
	Module      string
	Version     string
	Packages    []string
	License     string
	LicenseText string
	Notices     []Text
}

type Text struct {
	Name string
	Text string
}

// LoadPolicy accepts only the exact reviewed policy and verifies every
// repository-owned input before returning it.
func LoadPolicy(repoRoot string) (Policy, error) {
	data, err := readBoundedText(repoRoot, policyPath, policySHA256)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	dec := json.NewDecoder(bytes.NewBufferString(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Policy{}, fmt.Errorf("decode dependency notice policy: %w", err)
	}
	if err := requireJSONEOF(dec); err != nil {
		return Policy{}, err
	}
	if err := validateExactPolicy(p); err != nil {
		return Policy{}, err
	}
	p.inputs = make(map[string]string, len(p.Overrides)+2)
	for _, item := range p.Overrides {
		text, err := readBoundedText(repoRoot, item.LicensePath, item.LicenseSHA256)
		if err != nil {
			return Policy{}, fmt.Errorf("override %s@%s: %w", item.Module, item.Version, err)
		}
		p.inputs[item.LicensePath] = text
	}
	for _, item := range []struct{ path, digest string }{
		{p.Toolchain.LicensePath, p.Toolchain.LicenseSHA256},
		{p.Toolchain.PatentsPath, p.Toolchain.PatentsSHA256},
	} {
		text, err := readBoundedText(repoRoot, item.path, item.digest)
		if err != nil {
			return Policy{}, fmt.Errorf("toolchain %s: %w", item.path, err)
		}
		p.inputs[item.path] = text
	}
	return p, nil
}

func requireJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode dependency notice policy trailer: %w", err)
	}
	return errors.New("dependency notice policy contains multiple JSON values")
}

func validateExactPolicy(got Policy) error {
	got.inputs = nil
	want := expectedPolicy
	if got.SchemaVersion != want.SchemaVersion || got.GoLicenses != want.GoLicenses || got.Toolchain != want.Toolchain ||
		!slices.Equal(got.AllowedGoLicenses, want.AllowedGoLicenses) || !slices.Equal(got.Overrides, want.Overrides) {
		return errors.New("dependency notice policy differs from the reviewed exact tool, license, toolchain, or override contract")
	}
	return nil
}

func readBoundedText(repoRoot, rel, wantDigest string) (string, error) {
	if filepath.IsAbs(rel) || filepath.Clean(rel) != rel || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe repository-relative input %q", rel)
	}
	file, err := openRepositoryFileNoFollow(repoRoot, rel)
	if err != nil {
		return "", fmt.Errorf("open repository-relative input %s: %w", rel, err)
	}
	defer file.Close()
	return readOpenedBoundedText(file, rel, wantDigest)
}

func readOpenedBoundedText(file *os.File, rel, wantDigest string) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("inspect opened input %s: %w", rel, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("input %s is not a regular non-symlink file", rel)
	}
	if info.Size() > maxInputSize {
		return "", fmt.Errorf("input %s exceeds %d bytes", rel, maxInputSize)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxInputSize+1))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", rel, err)
	}
	if len(data) > maxInputSize {
		return "", fmt.Errorf("input %s exceeds %d bytes", rel, maxInputSize)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("input %s is not UTF-8", rel)
	}
	if wantDigest != "" {
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != wantDigest {
			return "", fmt.Errorf("input %s SHA-256 does not match the reviewed policy", rel)
		}
	}
	return string(data), nil
}

// DependencyFromLibrary converts the public go-licenses result without using
// its network-capable license-URL discovery path.
func DependencyFromLibrary(lib *golicense.Library, module, version, licenseText string, notices []Text) (Dependency, error) {
	if lib == nil {
		return Dependency{}, errors.New("go-licenses library is nil")
	}
	if len(lib.Licenses) != 1 {
		return Dependency{}, fmt.Errorf("library %q has %d classified licenses; expected exactly one", lib.Name(), len(lib.Licenses))
	}
	packages := slices.Clone(lib.Packages)
	sort.Strings(packages)
	packages = slices.Compact(packages)
	return Dependency{
		Library: lib.Name(), Module: module, Version: version, Packages: packages,
		License: lib.Licenses[0].Name, LicenseText: licenseText, Notices: slices.Clone(notices),
	}, nil
}

// Render produces stable complete text for reviewed Go dependency rows and
// the exact Go toolchain license and patent inputs. Browser-asset sections are
// appended by the C1b integration before THIRD_PARTY_NOTICES is replaced.
func Render(p Policy, dependencies []Dependency) ([]byte, error) {
	if err := validateExactPolicy(p); err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(p.AllowedGoLicenses))
	for _, name := range p.AllowedGoLicenses {
		allowed[name] = struct{}{}
	}
	items := slices.Clone(dependencies)
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		return a.Module+"\x00"+a.Version+"\x00"+a.Library < b.Module+"\x00"+b.Version+"\x00"+b.Library
	})
	var out strings.Builder
	out.WriteString("MARS Go Dependency Notices\n==========================\n\n")
	out.WriteString("Generated deterministically with " + p.GoLicenses.Module + "@" + p.GoLicenses.Version + ".\n\n")
	lastKey := ""
	for _, item := range items {
		if err := validateDependency(item, allowed); err != nil {
			return nil, err
		}
		key := item.Module + "@" + item.Version + "\x00" + item.Library
		if key == lastKey {
			return nil, fmt.Errorf("duplicate dependency notice row %s@%s (%s)", item.Module, item.Version, item.Library)
		}
		lastKey = key
		fmt.Fprintf(&out, "%s — %s\n%s\n\n", item.Module+"@"+item.Version, item.License, strings.Repeat("-", len(item.Module)+len(item.Version)+len(item.License)+4))
		out.WriteString("Library: " + item.Library + "\nPackages:\n")
		for _, pkg := range item.Packages {
			out.WriteString("- " + pkg + "\n")
		}
		out.WriteString("\nLicense:\n\n" + normalizeText(item.LicenseText) + "\n")
		notices := slices.Clone(item.Notices)
		sort.Slice(notices, func(i, j int) bool { return notices[i].Name < notices[j].Name })
		lastNoticeName := ""
		for _, notice := range notices {
			if strings.TrimSpace(notice.Name) == "" || !utf8.ValidString(notice.Name) || len(notice.Name) > maxInputSize ||
				strings.TrimSpace(notice.Text) == "" || !utf8.ValidString(notice.Text) || len(notice.Text) > maxInputSize {
				return nil, fmt.Errorf("dependency %s@%s has invalid NOTICE metadata", item.Module, item.Version)
			}
			if notice.Name == lastNoticeName {
				return nil, fmt.Errorf("dependency %s@%s has a duplicate NOTICE name", item.Module, item.Version)
			}
			lastNoticeName = notice.Name
			out.WriteString("Notice " + notice.Name + ":\n\n" + normalizeText(notice.Text) + "\n")
		}
	}
	license, okLicense := p.inputs[p.Toolchain.LicensePath]
	patents, okPatents := p.inputs[p.Toolchain.PatentsPath]
	if !okLicense || !okPatents {
		return nil, errors.New("verified Go toolchain license and patent inputs are unavailable")
	}
	toolchainName := "Go toolchain " + strings.TrimPrefix(p.Toolchain.Version, "go")
	out.WriteString(toolchainName + "\n" + strings.Repeat("-", len(toolchainName)) + "\n\nLicense:\n\n")
	out.WriteString(normalizeText(license) + "\nPatents:\n\n" + normalizeText(patents))
	return []byte(out.String()), nil
}

func validateDependency(item Dependency, allowed map[string]struct{}) error {
	if strings.TrimSpace(item.Library) == "" || strings.TrimSpace(item.Module) == "" || strings.TrimSpace(item.Version) == "" || len(item.Packages) == 0 {
		return errors.New("dependency notice row is missing library, module, version, or package identity")
	}
	if _, ok := allowed[item.License]; !ok {
		return fmt.Errorf("dependency %s@%s has unreviewed license %q", item.Module, item.Version, item.License)
	}
	if strings.TrimSpace(item.LicenseText) == "" || !utf8.ValidString(item.LicenseText) || len(item.LicenseText) > maxInputSize {
		return fmt.Errorf("dependency %s@%s has invalid or oversized license text", item.Module, item.Version)
	}
	packages := slices.Clone(item.Packages)
	sort.Strings(packages)
	packages = slices.Compact(packages)
	if !slices.Equal(packages, item.Packages) {
		return fmt.Errorf("dependency %s@%s packages are not sorted and unique", item.Module, item.Version)
	}
	return nil
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimRight(text, "\n") + "\n"
}
