/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/greaveselliott/mars-harness/internal/hardware"
)

const (
	defaultTierPortRange = 10
	portHealthTimeout    = 500 * time.Millisecond
	freshLockGrace       = 30 * time.Second
)

type PortConflictError struct {
	Port       int
	PID        int
	Tier       hardware.Tier
	Role       string
	Owner      string
	LockedTier hardware.Tier
	ModelPath  string
	ContextLen int
	Parallel   int
}

func (e *PortConflictError) Error() string {
	pid := "unknown"
	if e != nil && e.PID > 0 {
		pid = strconv.Itoa(e.PID)
	}
	tier := ""
	if e != nil {
		tier = string(e.Tier)
	}
	role := ""
	if e != nil {
		role = e.Role
	}
	port := 0
	if e != nil {
		port = e.Port
	}
	return fmt.Sprintf("inference_port_conflict: port=%d owning_pid=%s tier=%s role=%s remediation=stop the process using this port, wait for the previous mars-harness run to exit, or rerun with --model-endpoint <url>", port, pid, tier, role)
}

type portReservation struct {
	path string
	port int
}

type portReservationMetadata struct {
	Tier       hardware.Tier `json:"tier,omitempty"`
	ModelPath  string        `json:"model_path,omitempty"`
	ModelName  string        `json:"model_name,omitempty"`
	ContextLen int           `json:"context_len,omitempty"`
	Parallel   int           `json:"parallel,omitempty"`
}

type portLockInfo struct {
	Owner      string `json:"owner,omitempty"`
	PID        int    `json:"pid"`
	Port       int    `json:"port,omitempty"`
	Tier       string `json:"tier,omitempty"`
	ModelPath  string `json:"model_path,omitempty"`
	ModelName  string `json:"model_name,omitempty"`
	ContextLen int    `json:"context_len,omitempty"`
	Parallel   int    `json:"parallel,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

func (r *portReservation) Release() {
	if r == nil || strings.TrimSpace(r.path) == "" {
		return
	}
	_ = os.Remove(r.path)
	r.path = ""
}

func acquirePortReservation(port int, metas ...portReservationMetadata) (*portReservation, error) {
	if port <= 0 {
		return nil, fmt.Errorf("inference: invalid port %d", port)
	}
	var meta portReservationMetadata
	if len(metas) > 0 {
		meta = metas[0]
	}
	dir := filepath.Join(os.TempDir(), "mars-harness-inference-ports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("inference: create port lock dir: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.lock", port))
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = json.NewEncoder(f).Encode(portLockInfo{
				Owner:      "mars-harness",
				PID:        os.Getpid(),
				Port:       port,
				Tier:       string(meta.Tier),
				ModelPath:  strings.TrimSpace(meta.ModelPath),
				ModelName:  strings.TrimSpace(meta.ModelName),
				ContextLen: meta.ContextLen,
				Parallel:   meta.Parallel,
				CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			})
			_ = f.Close()
			return &portReservation{path: path, port: port}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("inference: acquire port lock %d: %w", port, err)
		}
		info := readPortLockInfo(path)
		if info.PID == 0 && freshLock(path) {
			return nil, &PortConflictError{
				Port:  port,
				Owner: "mars-harness",
			}
		}
		if info.PID <= 0 || !pidAlive(info.PID) {
			_ = os.Remove(path)
			continue
		}
		return nil, &PortConflictError{
			Port:       port,
			PID:        info.PID,
			Owner:      info.Owner,
			LockedTier: hardware.Tier(info.Tier),
			ModelPath:  info.ModelPath,
			ContextLen: info.ContextLen,
			Parallel:   info.Parallel,
		}
	}
}

func freshLock(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(st.ModTime()) < freshLockGrace
}

func readPortLockInfo(path string) portLockInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return portLockInfo{}
	}
	var info portLockInfo
	if err := json.Unmarshal(data, &info); err == nil && info.PID > 0 {
		return info
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return portLockInfo{}
	}
	return portLockInfo{PID: pid}
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func portAvailable(port int) (bool, int) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		_ = ln.Close()
		return true, 0
	}
	return false, owningPIDForPort(port)
}

func owningPIDForPort(port int) int {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}

func healthyLocalEndpoint(ctx context.Context, port int) bool {
	reqCtx, cancel := context.WithTimeout(ctx, portHealthTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", port), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
