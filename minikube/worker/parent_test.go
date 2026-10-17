package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"k8s.io/minikube/pkg/minikube/config"
)

// TestParentClientGetCrossProcess loads a profile fixture through a genuinely
// separate worker process: the test binary re-executed via TestMain.
func TestParentClientGetCrossProcess(t *testing.T) {
	home := isolatedMinikubeHome(t)
	writeProfileFixture(t, home, "india", nil)

	client := testClient(t)
	resp, err := client.Run(context.Background(), getProfileRequest("india"))
	require.NoError(t, err)
	require.True(t, resp.OK, "worker get failed: %s", resp.Error)
	require.Equal(t, "india", resp.ClusterConfig.Name)
	require.Equal(t, defaultTestKubernetesVersion, resp.ClusterConfig.KubernetesConfig.KubernetesVersion)
}

// TestParentClientProbeIsolationConcurrent is the process-boundary proof: many
// concurrent workers each set the process-global profile value the same way a
// start does, and each must observe its own value. In a single shared process,
// racing probes observe whichever writer ran last — the exact contamination
// that motivated worker isolation.
func TestParentClientProbeIsolationConcurrent(t *testing.T) {
	const workers = 4

	client := testClient(t)
	responses := make([]*Response, workers)
	errs := make([]error, workers)
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			profile := fmt.Sprintf("probe-worker-%d", i)
			resp, err := client.Run(context.Background(), &Request{Op: OpProbe, Probe: profile})
			if err != nil {
				errs[i] = err
				return
			}
			if !resp.OK {
				errs[i] = fmt.Errorf("worker probe failed: %s", resp.Error)
				return
			}
			if resp.Probe != profile {
				errs[i] = fmt.Errorf("settings leaked across workers: got %q, want %q", resp.Probe, profile)
				return
			}
			responses[i] = resp
		}()
	}
	wg.Wait()
	for i := range workers {
		require.NoError(t, errs[i], "probe worker %d failed", i)
		require.NotNil(t, responses[i])
	}
}

// TestParentClientConcurrentGetsDistinctHomes runs get ops against different
// disposable minikube homes concurrently; each worker resolves its own home,
// which also documents that workers inherit the parent environment.
func TestParentClientConcurrentGetsDistinctHomes(t *testing.T) {
	const workers = 2

	homes := make([]string, workers)
	mkHomes := make([]string, workers)
	for i := range homes {
		base, err := os.MkdirTemp("", "worker-home-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			os.RemoveAll(base)
		})
		homes[i] = base
		mkHomes[i] = filepath.Join(base, ".minikube")
		writeProfileFixture(t, mkHomes[i], "juliet", nil)
	}

	client := testClient(t)
	responses := make([]*Response, workers)
	errs := make([]error, workers)
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			c := *client
			c.ExtraEnv = append(c.ExtraEnv, "MINIKUBE_HOME="+homes[i])
			resp, err := c.Run(context.Background(), &Request{Op: OpGet, Profile: "juliet"})
			if err != nil {
				errs[i] = err
				return
			}
			if !resp.OK {
				errs[i] = fmt.Errorf("get failed: %s", resp.Error)
				return
			}
			responses[i] = resp
		}()
	}
	wg.Wait()
	for i := range workers {
		require.NoError(t, errs[i], "worker %d failed", i)

		// The saved fixture's version distinguishes the homes.
		want := fixtureKubernetesVersion(t, mkHomes[i])
		require.NotNil(t, responses[i])
		require.Equal(t, want, responses[i].ClusterConfig.KubernetesConfig.KubernetesVersion)
	}
}

func fixtureKubernetesVersion(t *testing.T, mkHome string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(mkHome, "profiles", "juliet", "config.json"))
	require.NoError(t, err)
	var cc config.ClusterConfig
	require.NoError(t, json.Unmarshal(b, &cc))
	return cc.KubernetesConfig.KubernetesVersion
}

// TestParentClientCancellationTimeout verifies Terraform cancellation
// propagation: a stuck worker is terminated (with its process group) and the
// parent surfaces an actionable diagnostic instead of hanging.
func TestParentClientCancellationTimeout(t *testing.T) {
	isolatedMinikubeHome(t)

	client := testClient(t, EnvTestDelayMs+"=20000")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.Run(ctx, getProfileRequest("whatever"))
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Contains(t, err.Error(), "cancelled")
	require.Less(t, elapsed, 5*time.Second, "the stuck worker should have been killed, not awaited")
}

// TestParentClientMalformedResponse classifies a worker that fails protocol
// framing, and proves the stderr tail is attached to the parent's diagnostics.
func TestParentClientMalformedResponse(t *testing.T) {
	isolatedMinikubeHome(t)

	client := testClient(t, EnvTestGarbage+"=1")
	_, err := client.Run(context.Background(), getProfileRequest("kilo"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "malformed worker output")
	require.Contains(t, err.Error(), testStderrSentinel, "diagnostics should include the worker stderr tail")
}

// TestParentClientAbnormalExit classifies upstream process termination (the
// exit.Error path) as an abnormal worker exit with the stderr tail attached.
func TestParentClientAbnormalExit(t *testing.T) {
	isolatedMinikubeHome(t)

	client := testClient(t, EnvTestExit+"=7")
	_, err := client.Run(context.Background(), getProfileRequest("lima"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "exited abnormally")
	require.Contains(t, err.Error(), "7")
	require.Contains(t, err.Error(), testStderrSentinel)
}

// TestParentClientUnknownOperation proves op errors travel the protocol as a
// typed non-OK response rather than a process failure.
func TestParentClientUnknownOperation(t *testing.T) {
	isolatedMinikubeHome(t)

	client := testClient(t)
	resp, err := client.Run(context.Background(), &Request{Op: "teleport", Profile: "mike"})
	require.NoError(t, err, "typed protocol errors must not surface as process failures")
	require.False(t, resp.OK)
	require.Contains(t, resp.Error, "unknown operation")
}

// TestProfileLockSerializesSameProfile exercises the keyed lock directly: a
// same-profile operation waits, a distinct profile does not.
func TestProfileLockSerializesSameProfile(t *testing.T) {
	release, err := lockProfile("november")
	require.NoError(t, err)

	reacquired := make(chan struct{}, 1)
	go func() {
		r, e := lockProfile("november")
		if e != nil {
			t.Errorf("lockProfile(november) failed: %v", e)
			return
		}
		r()
		reacquired <- struct{}{}
	}()

	select {
	case <-reacquired:
		t.Fatal("same profile lock was acquired while held")
	case <-time.After(100 * time.Millisecond):
		// still held, as expected
	}

	release()
	require.Eventually(t, func() bool {
		select {
		case <-reacquired:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "same profile lock should become available after release")

	// Distinct profile: acquisition succeeds immediately even while the other
	// goroutine's "november" lock exists.
	other, err := lockProfile("oscar")
	require.NoError(t, err)
	other()
}
