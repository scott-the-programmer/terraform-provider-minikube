package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"k8s.io/klog/v2"
	cmdcfg "k8s.io/minikube/cmd/minikube/cmd/config"
	minikubeAddons "k8s.io/minikube/pkg/addons"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/kubeconfig"
	"k8s.io/minikube/pkg/minikube/localpath"
	"k8s.io/minikube/pkg/minikube/run"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/lib"
)

func IsWorkerChild() bool {
	return os.Getenv(EnvWorkerChild) == "1"
}

// Serve replies to one request with a frame and an exit code.
func Serve(in io.Reader, out io.Writer) int {
	klog.SetOutput(os.Stderr)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)

	write := func(resp *Response, code int) int {
		if err := writeFrame(out, resp); err != nil {
			fmt.Fprintf(os.Stderr, "worker: writing response: %v\n", err)
			return 1
		}
		return code
	}

	raw, err := io.ReadAll(in)
	if err != nil {
		return write(&Response{Op: OpInvalid, Error: fmt.Sprintf("reading worker request: %v", err)}, 1)
	}

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return write(&Response{Op: OpInvalid, Error: fmt.Sprintf("decoding worker request: %v", err)}, 1)
	}

	if err := applyTestSeams(); err != nil {
		return write(&Response{Op: req.Op, Error: err.Error()}, 0)
	}

	return write(handle(req), 0)
}

func applyTestSeams() error {
	if v := os.Getenv(EnvTestDelayMs); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid %s %q: %v", EnvTestDelayMs, v, err)
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	return nil
}

func handle(req Request) *Response {
	ctx, cancel := opContext(req.Deadline)
	defer cancel()

	resp := &Response{Op: req.Op}

	switch req.Op {
	case OpStart:
		kc, err := startClient(req).Start()
		if err != nil {
			resp.Error = fmt.Sprintf("start cluster %q: %v", req.Profile, err)
			return resp
		}
		resp.OK = true
		resp.Kubeconfig = outputsFromKubeconfig(kc)
		return resp

	case OpAddons:
		if err := applyAddons(ctx, req.Profile, req.Enable, req.Disable); err != nil {
			resp.Error = fmt.Sprintf("apply addons on cluster %q: %v", req.Profile, err)
			return resp
		}
		resp.OK = true
		return resp

	case OpGet:
		// Use config.Load because mustload.Partial exits on missing profiles.
		saved, err := config.Load(req.Profile)
		if err != nil {
			if config.IsNotExist(err) {
				resp.Error = fmt.Sprintf("profile %q not found in minikube home %q", req.Profile, localpath.MiniPath())
				return resp
			}
			resp.Error = fmt.Sprintf("load profile %q: %v", req.Profile, err)
			return resp
		}
		resp.OK = true
		resp.ClusterConfig = saved
		return resp

	case OpDelete:
		if err := deleteClient(req).Delete(); err != nil {
			resp.Error = fmt.Sprintf("delete cluster %q: %v", req.Profile, err)
			return resp
		}
		resp.OK = true
		return resp

	case OpProbe:
		viper.Set(cmdcfg.Bootstrapper, "kubeadm")
		viper.Set(config.ProfileName, req.Probe)
		viper.Set("preload", true)
		resp.OK = true
		resp.Probe = viper.GetString(config.ProfileName)
		return resp

	default:
		resp.Error = fmt.Sprintf("unknown operation %q", req.Op)
		return resp
	}
}

func opContext(deadline time.Time) (context.Context, context.CancelFunc) {
	if deadline.IsZero() {
		return context.WithCancel(context.Background())
	}
	return context.WithDeadline(context.Background(), deadline)
}

func startClient(req Request) *lib.MinikubeClient {
	return lib.NewMinikubeClient(*req.Config, lib.MinikubeClientDeps{
		Node:       lib.NewMinikubeCluster(),
		Downloader: lib.NewMinikubeDownloader(),
	})
}

func deleteClient(req Request) *lib.MinikubeClient {
	cfg := &config.ClusterConfig{}
	if req.Config != nil && req.Config.ClusterConfig != nil {
		cfg = req.Config.ClusterConfig
	}
	return lib.NewMinikubeClient(lib.MinikubeClientConfig{
		ClusterConfig: cfg,
		ClusterName:   req.Profile,
	}, lib.MinikubeClientDeps{
		Node: lib.NewMinikubeCluster(),
	})
}

// Send output fields only; kubeconfig.Settings contains an atomic.Value.
func outputsFromKubeconfig(kc *kubeconfig.Settings) *KubeconfigOutputs {
	return &KubeconfigOutputs{
		ClusterName:          kc.ClusterName,
		ClusterServerAddress: kc.ClusterServerAddress,
		ClientKey:            kc.ClientKey,
		ClientCertificate:    kc.ClientCertificate,
		CertificateAuthority: kc.CertificateAuthority,
	}
}

func applyAddons(ctx context.Context, profile string, enable, disable []string) error {
	options := &run.CommandOptions{NonInteractive: true}
	if err := lib.ResetAddonAssets(); err != nil {
		return err
	}
	_ = ctx // addon writers are synchronous; the deadline applies before dispatch
	for _, addon := range enable {
		if err := minikubeAddons.SetAndSave(profile, addon, "true", options); err != nil {
			return fmt.Errorf("enable addon %s: %w", addon, err)
		}
	}
	for _, addon := range disable {
		if err := minikubeAddons.SetAndSave(profile, addon, "false", options); err != nil {
			return fmt.Errorf("disable addon %s: %w", addon, err)
		}
	}
	return nil
}
