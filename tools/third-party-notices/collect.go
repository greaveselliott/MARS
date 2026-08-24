package notices

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	golicense "github.com/google/go-licenses/v2/licenses"
	"k8s.io/klog/v2"
)

const (
	mainModule       = "github.com/greaveselliott/mars"
	requiredGo       = "go1.27.0"
	noticesPath      = "THIRD_PARTY_NOTICES"
	collectionWindow = 2 * time.Minute
)

var (
	supportedTargets = []target{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}
	noticeName       = regexp.MustCompile(`^NOTICE(\.(txt|md))?$`)
)

type target struct{ GOOS, GOARCH string }

type goListPackage struct {
	ImportPath string
	Standard   bool
	Module     *goListModule
}

type goListModule struct {
	Path    string
	Version string
	Dir     string
	Main    bool
	Replace *goListModule
}

// Generate collects and reconciles the exact union of dependencies linked by
// the four supported release builds, then renders the complete checked-in
// notice package. It uses the pinned go-licenses library API and never invokes
// the network-capable report command.
func Generate(repoRoot string) ([]byte, error) {
	if runtime.Version() != requiredGo || os.Getenv("GOROOT") == "" || build.Default.GOROOT == "" || filepath.Clean(os.Getenv("GOROOT")) != filepath.Clean(build.Default.GOROOT) {
		return nil, fmt.Errorf("toolchain_root_invalid: run the notice check with exact %s and GOROOT=$(%s env GOROOT)", requiredGo, "go")
	}
	p, err := LoadPolicy(repoRoot)
	if err != nil {
		return nil, err
	}
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve notice-generator working directory: %w", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		return nil, fmt.Errorf("enter MARS repository for dependency collection: %w", err)
	}
	defer func() { _ = os.Chdir(oldWorkingDirectory) }()
	ctx, cancel := context.WithTimeout(context.Background(), collectionWindow)
	defer cancel()
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)

	byKey := map[string]Dependency{}
	overridesUsed := map[string]bool{}
	for _, lane := range supportedTargets {
		rows, used, err := collectTarget(ctx, repoRoot, p, lane)
		if err != nil {
			return nil, fmt.Errorf("collect %s/%s: %w", lane.GOOS, lane.GOARCH, err)
		}
		for key := range used {
			overridesUsed[key] = true
		}
		for _, row := range rows {
			key := dependencyKey(row)
			if prior, ok := byKey[key]; ok {
				if prior.License != row.License || normalizeText(prior.LicenseText) != normalizeText(row.LicenseText) || !slices.Equal(prior.Notices, row.Notices) {
					return nil, fmt.Errorf("dependency notice differs across supported release targets: %s@%s", row.Module, row.Version)
				}
				prior.Packages = append(prior.Packages, row.Packages...)
				sort.Strings(prior.Packages)
				prior.Packages = slices.Compact(prior.Packages)
				byKey[key] = prior
				continue
			}
			byKey[key] = row
		}
	}
	if err := requireOverridesUsed(p, overridesUsed); err != nil {
		return nil, err
	}
	rows := make([]Dependency, 0, len(byKey))
	for _, row := range byKey {
		rows = append(rows, row)
	}
	return RenderThirdPartyNotices(p, rows)
}

func requireOverridesUsed(p Policy, used map[string]bool) error {
	if len(used) != len(p.Overrides) {
		return errors.New("reviewed dependency-license override set is incomplete, stale, or contains an unmatched entry")
	}
	for _, override := range p.Overrides {
		key := override.Module + "@" + override.Version
		if !used[key] {
			return fmt.Errorf("reviewed dependency-license override is stale or unused: %s", key)
		}
	}
	return nil
}

func collectTarget(ctx context.Context, repoRoot string, p Policy, lane target) ([]Dependency, map[string]bool, error) {
	restore := setTargetEnvironment(lane)
	defer restore()

	packages, err := listTargetPackages(ctx, repoRoot)
	if err != nil {
		return nil, nil, err
	}
	if len(packages) == 0 {
		return nil, nil, errors.New("independent go list returned no packages")
	}
	classifier, err := golicense.NewClassifier()
	if err != nil {
		return nil, nil, fmt.Errorf("initialize pinned license classifier: %w", err)
	}
	libraries, err := golicense.Libraries(ctx, classifier, false, nil, "./cmd/mars")
	if err != nil {
		return nil, nil, fmt.Errorf("collect pinned go-licenses rows: %w", err)
	}
	if len(libraries) == 0 {
		return nil, nil, errors.New("pinned go-licenses returned no rows")
	}

	overrides := make(map[string]Override, len(p.Overrides))
	for _, item := range p.Overrides {
		overrides[item.Module+"@"+item.Version] = item
	}
	used := map[string]bool{}
	covered := map[string]bool{}
	rows := make([]Dependency, 0, len(libraries))
	for _, lib := range libraries {
		module, external, err := libraryModule(lib, packages)
		if err != nil {
			return nil, nil, err
		}
		if !external {
			continue
		}
		for _, pkg := range lib.Packages {
			if covered[pkg] {
				return nil, nil, fmt.Errorf("external package appears in multiple license rows: %s", pkg)
			}
			covered[pkg] = true
		}
		key := module.Path + "@" + module.Version
		var licenseName, licenseText string
		var notices []Text
		switch len(lib.Licenses) {
		case 0:
			override, ok := overrides[key]
			if !ok {
				return nil, nil, fmt.Errorf("dependency has an unknown license with no reviewed override: %s", key)
			}
			licenseName = override.License
			licenseText = p.inputs[override.LicensePath]
			used[key] = true
		case 1:
			if _, stale := overrides[key]; stale {
				return nil, nil, fmt.Errorf("reviewed override conflicts with a recognized license: %s", key)
			}
			licenseName = lib.Licenses[0].Name
			licenseText, err = readExternalText(lib.LicenseFile)
			if err != nil {
				return nil, nil, fmt.Errorf("read license for %s: %w", key, err)
			}
			notices, err = readCompanionNotices(lib.LicenseFile)
			if err != nil {
				return nil, nil, fmt.Errorf("read notices for %s: %w", key, err)
			}
		default:
			names := make([]string, 0, len(lib.Licenses))
			for _, license := range lib.Licenses {
				names = append(names, license.Name)
			}
			sort.Strings(names)
			names = slices.Compact(names)
			for _, name := range names {
				if !slices.Contains(p.AllowedGoLicenses, name) {
					return nil, nil, fmt.Errorf("dependency has a forbidden or unknown license classification: %s (%s)", key, name)
				}
			}
			licenseName = strings.Join(names, " AND ")
			licenseText, err = readExternalText(lib.LicenseFile)
			if err != nil {
				return nil, nil, fmt.Errorf("read multi-license text for %s: %w", key, err)
			}
			notices, err = readCompanionNotices(lib.LicenseFile)
			if err != nil {
				return nil, nil, fmt.Errorf("read notices for %s: %w", key, err)
			}
		}
		row := Dependency{
			Library: lib.Name(), Module: module.Path, Version: module.Version,
			Identity: stableLicenseIdentity(lib, module),
			Packages: slices.Clone(lib.Packages), License: licenseName,
			LicenseText: licenseText, Notices: notices,
		}
		sort.Strings(row.Packages)
		row.Packages = slices.Compact(row.Packages)
		rows = append(rows, row)
	}

	externalCount := 0
	for path, module := range packages {
		if module.Path == mainModule {
			continue
		}
		externalCount++
		if !covered[path] {
			return nil, nil, fmt.Errorf("external package is absent from go-licenses rows: %s", path)
		}
	}
	if externalCount == 0 || len(rows) == 0 {
		return nil, nil, errors.New("external dependency inventory is empty")
	}
	return rows, used, nil
}

func setTargetEnvironment(lane target) func() {
	keys := []string{"GOOS", "GOARCH", "CGO_ENABLED", "GOTOOLCHAIN", "GOPROXY", "GOSUMDB", "GOVCS", "GOAUTH", "GOWORK", "GOFLAGS"}
	old := make(map[string]*string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			copy := value
			old[key] = &copy
		} else {
			old[key] = nil
		}
	}
	values := map[string]string{
		"GOOS": lane.GOOS, "GOARCH": lane.GOARCH, "CGO_ENABLED": "0",
		"GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
		"GOVCS": "*:off", "GOAUTH": "off", "GOWORK": "off",
		"GOFLAGS": "-mod=readonly -buildvcs=false",
	}
	for key, value := range values {
		_ = os.Setenv(key, value)
	}
	return func() {
		for _, key := range keys {
			if old[key] == nil {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, *old[key])
			}
		}
	}
}

func listTargetPackages(ctx context.Context, repoRoot string) (map[string]goListModule, error) {
	command := exec.CommandContext(ctx, filepath.Join(build.Default.GOROOT, "bin", "go"), "list", "-deps", "-json", "-mod=readonly", "./cmd/mars")
	command.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("independent go list failed: %w", err)
	}
	if stderr.Len() != 0 {
		return nil, errors.New("independent go list emitted unexpected diagnostics")
	}
	dec := json.NewDecoder(&stdout)
	result := map[string]goListModule{}
	for {
		var pkg goListPackage
		err := dec.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode independent go list: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Module == nil {
			return nil, fmt.Errorf("non-standard package has no module identity: %s", pkg.ImportPath)
		}
		module := *pkg.Module
		if module.Replace != nil {
			module = *module.Replace
		}
		if prior, ok := result[pkg.ImportPath]; ok && (prior.Path != module.Path || prior.Version != module.Version) {
			return nil, fmt.Errorf("package has conflicting module identities: %s", pkg.ImportPath)
		}
		result[pkg.ImportPath] = module
	}
	return result, nil
}

func libraryModule(lib *golicense.Library, packages map[string]goListModule) (goListModule, bool, error) {
	if lib == nil || len(lib.Packages) == 0 {
		return goListModule{}, false, errors.New("go-licenses returned an empty library row")
	}
	var selected goListModule
	for index, path := range lib.Packages {
		module, ok := packages[path]
		if !ok {
			return goListModule{}, false, fmt.Errorf("go-licenses package is absent from independent go list: %s", path)
		}
		if index == 0 {
			selected = module
		} else if selected.Path != module.Path || selected.Version != module.Version {
			return goListModule{}, false, fmt.Errorf("go-licenses row spans multiple modules: %s", lib.Name())
		}
	}
	return selected, selected.Path != mainModule, nil
}

func readExternalText(path string) (string, error) {
	if path == "" {
		return "", errors.New("license path is empty")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return readOpenedBoundedText(file, filepath.Base(path), "")
}

func readCompanionNotices(licensePath string) ([]Text, error) {
	entries, err := os.ReadDir(filepath.Dir(licensePath))
	if err != nil {
		return nil, err
	}
	var result []Text
	for _, entry := range entries {
		if entry.IsDir() || !noticeName.MatchString(entry.Name()) || entry.Name() == filepath.Base(licensePath) {
			continue
		}
		text, err := readExternalText(filepath.Join(filepath.Dir(licensePath), entry.Name()))
		if err != nil {
			return nil, err
		}
		result = append(result, Text{Name: entry.Name(), Text: text})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func dependencyKey(item Dependency) string {
	return item.Module + "@" + item.Version + "\x00" + item.Library + "\x00" + item.Identity
}

func stableLicenseIdentity(lib *golicense.Library, module goListModule) string {
	if lib.LicenseFile != "" && module.Dir != "" {
		if relative, err := filepath.Rel(module.Dir, lib.LicenseFile); err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(relative)
		}
	}
	packages := slices.Clone(lib.Packages)
	sort.Strings(packages)
	return "packages:" + strings.Join(packages, ",")
}

func RenderThirdPartyNotices(p Policy, dependencies []Dependency) ([]byte, error) {
	goNotices, err := Render(p, dependencies)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("MARS Third-Party Notices\n========================\n\n")
	out.WriteString("This file contains the license and notice text for third-party material distributed in MARS release archives.\n\n")
	for _, asset := range p.BrowserAssets {
		text, ok := p.inputs[asset.Path]
		if !ok {
			return nil, fmt.Errorf("verified browser license is unavailable: %s", asset.Name)
		}
		heading := asset.Name + " — " + asset.License
		out.WriteString(heading + "\n" + strings.Repeat("-", len(heading)) + "\n\n")
		out.WriteString(normalizeText(text) + "\n")
	}
	out.Write(goNotices)
	result := []byte(out.String())
	if bytes.Contains(bytes.ToLower(result), []byte("provisional")) {
		return nil, errors.New("generated notices retain provisional wording")
	}
	return result, nil
}

func Digest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
