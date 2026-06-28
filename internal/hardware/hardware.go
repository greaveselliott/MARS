/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Profile represents the detected hardware tier.
type Profile string

const (
	ProfileCPU    Profile = "cpu"
	ProfileLow    Profile = "low"    // <8GB VRAM
	ProfileMedium Profile = "medium" // 8-16GB VRAM
	ProfileHigh   Profile = "high"   // 16-48GB VRAM
	ProfileMulti  Profile = "multi"  // Multiple GPUs
)

// GPU holds detected GPU information.
type GPU struct {
	Index   int
	Name    string
	VRAMMiB int
	Driver  string
}

// Summary is the detected hardware of the current machine.
type Summary struct {
	Profile  Profile
	GPUs     []GPU
	RAMMiB   int
	CPUCores int
	OS       string
	Arch     string
}

// Detect queries the current machine's hardware.
// On failure to detect GPU, falls back to ProfileCPU without error.
func Detect() Summary {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	cores := runtime.NumCPU()
	if cores < 1 {
		cores = 1
	}

	ramMiB := detectRAMMiB(osName)
	gpus := detectGPUs(osName)
	profile := selectProfile(gpus)

	s := Summary{
		Profile:  profile,
		GPUs:     gpus,
		RAMMiB:   ramMiB,
		CPUCores: cores,
		OS:       osName,
		Arch:     arch,
	}

	slog.Info("hardware detected",
		"profile", string(profile),
		"gpu_count", len(gpus),
		"ram_mib", ramMiB,
		"cpu_cores", cores,
		"os", osName,
		"arch", arch,
	)
	return s
}

// selectProfile maps detected GPUs to a hardware profile.
func selectProfile(gpus []GPU) Profile {
	switch len(gpus) {
	case 0:
		return ProfileCPU
	case 1:
		v := gpus[0].VRAMMiB
		if v < 8192 {
			return ProfileLow
		}
		if v < 16384 {
			return ProfileMedium
		}
		return ProfileHigh
	default:
		return ProfileMulti
	}
}

func detectRAMMiB(goos string) int {
	switch goos {
	case "linux":
		return memTotalLinux()
	case "darwin":
		return memTotalDarwin()
	default:
		if m := memTotalLinux(); m > 0 {
			return m
		}
		slog.Warn("RAM detection unsupported for OS; reporting 0 MiB", "os", goos)
		return 0
	}
}

func memTotalLinux() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		slog.Debug("read /proc/meminfo failed", "err", err)
		return 0
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0
			}
			kb, err := strconv.Atoi(fields[1])
			if err != nil {
				slog.Debug("parse MemTotal failed", "err", err)
				return 0
			}
			return kb / 1024 // KiB -> MiB
		}
	}
	return 0
}

func memTotalDarwin() int {
	if mib := memTotalDarwinSyscall(); mib > 0 {
		return mib
	}
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		slog.Debug("sysctl hw.memsize failed", "err", err)
		return 0
	}
	s := strings.TrimSpace(string(out))
	bytesTotal, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		slog.Debug("parse hw.memsize failed", "err", err)
		return 0
	}
	return int(bytesTotal / (1024 * 1024))
}

func detectGPUs(goos string) []GPU {
	if g := tryNvidiaSMI(); len(g) > 0 {
		return g
	}
	if goos == "darwin" {
		if g := tryMacOSMetal(); len(g) > 0 {
			return g
		}
	}
	return nil
}

func tryNvidiaSMI() []GPU {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total,driver_version",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		slog.Debug("nvidia-smi unavailable or failed", "err", err)
		return nil
	}
	r := csv.NewReader(bytes.NewReader(out))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		slog.Debug("parse nvidia-smi csv failed", "err", err)
		return nil
	}
	var gpus []GPU
	for _, rec := range records {
		if len(rec) < 4 {
			continue
		}
		idx, err1 := strconv.Atoi(strings.TrimSpace(rec[0]))
		vram, err2 := strconv.Atoi(strings.TrimSpace(rec[2]))
		if err1 != nil || err2 != nil {
			continue
		}
		gpus = append(gpus, GPU{
			Index:   idx,
			Name:    strings.TrimSpace(rec[1]),
			VRAMMiB: vram,
			Driver:  strings.TrimSpace(rec[3]),
		})
	}
	if len(gpus) == 0 {
		slog.Debug("nvidia-smi produced no parsable GPU rows")
	}
	return gpus
}

var vramLineRE = regexp.MustCompile(`(?i)vram\D*(\d+)\s*(mb|gb)`)

func tryMacOSMetal() []GPU {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		slog.Debug("system_profiler failed", "err", err)
		return nil
	}
	text := string(out)
	if !strings.Contains(strings.ToLower(text), "metal") {
		slog.Debug("no Metal support reported in SPDisplaysDataType")
		return nil
	}

	matches := vramLineRE.FindAllStringSubmatch(text, -1)
	if len(matches) > 0 {
		var gpus []GPU
		for i, m := range matches {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			unit := strings.ToLower(m[2])
			vramMiB := n
			if unit == "gb" {
				vramMiB = n * 1024
			}
			gpus = append(gpus, GPU{
				Index:   i,
				Name:    "Metal",
				VRAMMiB: vramMiB,
				Driver:  "Metal",
			})
		}
		if len(gpus) > 0 {
			slog.Info("parsed Metal display VRAM", "gpu_count", len(gpus))
			return gpus
		}
	}

	// Apple Silicon uses unified memory — the GPU can access all system RAM.
	// Use sysctl hw.memsize to report accurate VRAM for profile selection.
	unifiedMiB := memTotalDarwin()
	if unifiedMiB <= 0 {
		unifiedMiB = 8192
	}
	slog.Info("Apple Silicon Metal: using unified memory as VRAM", "mib", unifiedMiB)
	return []GPU{{
		Index:   0,
		Name:    "Apple Silicon",
		VRAMMiB: unifiedMiB,
		Driver:  "Metal",
	}}
}
