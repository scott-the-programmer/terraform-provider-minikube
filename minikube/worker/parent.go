package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/minikube/pkg/minikube/localpath"
)

// Client launches one isolated worker process per operation. Create one with
// NewClient.
type Client struct {
	// ExecPath is the executable to re-execute. It defaults to
	// os.Executable(), so workers always run the real provider binary, which
	// reports "unknown resource and state behaviour" for nothing else.
	ExecPath string
	// ExtraEnv appends to the child environment (os.Environ() plus these).
	// The default client sets EnvWorkerChild so main dispatches worker mode
	// before plugin serving.
	ExtraEnv []string
	// StderrTail caps how much worker stderr is attached to diagnostics.
	StderrTail int
}

// NewClient returns a parent client that re-executes this provider binary in
// worker child mode.
func NewClient() (*Client, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve provider executable for worker mode: %w", err)
	}
	return &Client{
		ExecPath:   exe,
		ExtraEnv:   []string{fmt.Sprintf("%s=1", EnvWorkerChild)},
		StderrTail: 8192,
	}, nil
}

// IsolationEnabled reports whether lifecycle operations should run in isolated
// worker processes. Practitioners disable isolation by setting EnvInProcess,
// and the setting also reverts safety to the broad creation mutex, which
// serializes every cluster operation in-process. Worker processes never nest
// further workers.
func IsolationEnabled() bool {
	if IsWorkerChild() {
		return false
	}
	return os.Getenv(EnvInProcess) == ""
}

// Run executes one operation in a fresh worker process and returns its typed
// response. The child is always reaped before Run returns, so failed workers
// never linger.
//
// Operations on the same canonical Minikube home/profile serialize on a keyed
// lock, while distinct profiles may overlap. This is the replacement for the
// former broad creation mutex: coordination is scoped to each profile, and
// fresh per-worker processes stop Minikube's process-global Viper settings
// from crossing operations. Minikube's own file locks remain responsible for
// shared on-disk resources (kubeconfig, certificates, download caches).
func (c *Client) Run(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("nil worker request")
	}
	release, err := lockProfile(req.Profile)
	if err != nil {
		return nil, err
	}
	defer release()

	// Propagate deadlines: the finer of the request's own cutoff and the
	// caller's context applies both here (process-group kill) and in the
	// child (own deadline, absorbed into the request).
	deadline := req.Deadline
	if d, ok := ctx.Deadline(); ok && (deadline.IsZero() || d.Before(deadline)) {
		deadline = d
	}
	req.Deadline = deadline

	payload, err := encode(req)
	if err != nil {
		return nil, err
	}

	exe := c.ExecPath
	if exe == "" {
		var e error
		exe, e = os.Executable()
		if e != nil {
			return nil, fmt.Errorf("resolve provider executable for worker mode: %w", e)
		}
	}

	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(), c.ExtraEnv...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.SysProcAttr = childProcessAttr()
	var stdout bytes.Buffer
	var stderr capBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error {
		// Terraform cancelled or the deadline passed: take the entire worker
		// process group down so driver-spawned descendants do not outlive the
		// operation.
		if err := killProcessGroup(cmd); err != nil {
			if cmdErr := cmd.Process.Kill(); cmdErr != nil {
				klog.Errorf("worker: terminate process group: %v (fallback kill: %v)", err, cmdErr)
			}
		}
		return nil
	}

	if err := cmd.Start(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("worker cancelled before start: %w", ctx.Err())
		}
		return nil, fmt.Errorf("start worker process: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("worker did not survive operation %s on profile %q (cancelled or timed out: %v): %s",
				req.Op, req.Profile, ctx.Err(), tailString(stderr, c.StderrTail))
		}
		// The worker may have terminated upstream-style (exit.Error kills the
		// process without a diagnostic channel) after writing a valid frame;
		// parse before treating the exit status as fatal.
		if resp, perr := decodeFrame(stdout.Bytes()); perr == nil {
			return resp, nil
		}
		return nil, fmt.Errorf("worker exited abnormally during operation %s on profile %q (%v): %s",
			req.Op, req.Profile, err, tailString(stderr, c.StderrTail))
	}

	resp, perr := decodeFrame(stdout.Bytes())
	if perr != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("worker cancelled during operation %s on profile %q (%v): %s",
				req.Op, req.Profile, ctx.Err(), tailString(stderr, c.StderrTail))
		}
		return nil, fmt.Errorf("malformed worker output during operation %s on profile %q: %v: %s",
			req.Op, req.Profile, perr, tailString(stderr, c.StderrTail))
	}
	return resp, nil
}

// tailString renders stderr as a bounded diagnostic tail.
func tailString(b capBuffer, n int) string {
	raw := b.buf.Bytes()
	if n <= 0 || len(raw) <= n {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(string(raw[len(raw)-n:]))
}

// capBuffer keeps only the most recent bytes so a chatty worker's logs stay
// bounded in memory while still providing a useful tail on failure.
type capBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *capBuffer) Write(p []byte) (int, error) {
	n, err := b.buf.Write(p)
	if b.max > 0 && b.buf.Len() > b.max {
		kept := b.buf.Bytes()
		keeps := kept[len(kept)-b.max:]
		contended := make([]byte, len(keeps))
		copy(contended, keeps)
		b.buf.Reset()
		b.buf.Write(contended)
	}
	return n, err
}

// minikubeProfileKey identifies one Minikube profile inside one canonical
// Minikube home, so overlapping Terraform processes or provider
// configurations coordinate consistently.
func minikubeProfileKey(home, profile string) string {
	return fmt.Sprintf("%s\x00%s", home, profile)
}

// profileLock serializes operations that act on the same canonical Minikube
// home/profile within the provider process, while leaving different profiles
// free to overlap.
type profileLock struct {
	mu sync.Mutex
}

//nolint:gochecknoglobals // process-wide registry of per-profile coordination locks
var profileLockRegistry sync.Map // key string -> *profileLock

// lockProfile acquires the per-profile lock and returns its release func.
// Diagnostic operations that carry no profile (probe-isolation) deliberately
// skip coordination so they can overlap: every operation that mutates or reads
// a profile carries its name, and those serialize.
func lockProfile(profile string) (func(), error) {
	if strings.TrimSpace(profile) == "" {
		return func() {}, nil
	}
	v, _ := profileLockRegistry.LoadOrStore(minikubeProfileKey(localpath.MiniPath(), profile), &profileLock{})
	lock := v.(*profileLock)
	lock.mu.Lock()
	return lock.mu.Unlock, nil
}
