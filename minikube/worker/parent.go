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

type Client struct {
	// ExecPath defaults to os.Executable() when empty.
	ExecPath   string
	ExtraEnv   []string
	StderrTail int
}

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

func IsolationEnabled() bool {
	if IsWorkerChild() {
		return false
	}
	return os.Getenv(EnvInProcess) == ""
}

// Run reaps one worker and serializes calls per home/profile.
func (c *Client) Run(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("nil worker request")
	}
	release, err := lockProfile(req.Profile)
	if err != nil {
		return nil, err
	}
	defer release()

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
		// Prefer a framed response over the exit status.
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

func tailString(b capBuffer, n int) string {
	raw := b.buf.Bytes()
	if n <= 0 || len(raw) <= n {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(string(raw[len(raw)-n:]))
}

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

func minikubeProfileKey(home, profile string) string {
	return fmt.Sprintf("%s\x00%s", home, profile)
}

type profileLock struct {
	mu sync.Mutex
}

//nolint:gochecknoglobals // per-profile locks must be process-wide
var profileLockRegistry sync.Map // string -> *profileLock

func lockProfile(profile string) (func(), error) {
	if strings.TrimSpace(profile) == "" {
		return func() {}, nil
	}
	v, _ := profileLockRegistry.LoadOrStore(minikubeProfileKey(localpath.MiniPath(), profile), &profileLock{})
	lock := v.(*profileLock)
	lock.mu.Lock()
	return lock.mu.Unlock, nil
}
