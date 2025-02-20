//go:generate go run github.com/golang/mock/mockgen -source=$GOFILE -destination=mock_minikube_binary.go -package=$GOPACKAGE
package generator

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MinikubeBinary interface {
	GetVersion(ctx context.Context) (string, error)
	GetStartHelpText(ctx context.Context) (string, error)
}

type MinikubeHostBinary struct {
}

func (m *MinikubeHostBinary) GetVersion(ctx context.Context) (string, error) {
	return run(ctx, "version")
}

func (m *MinikubeHostBinary) GetStartHelpText(ctx context.Context) (string, error) {
	return run(ctx, "start", "--help")
}

var computedFields []string = []string{
	"apiserver_ips",
	"apiserver_names",
	"hyperkit_vsock_ports",
	"insecure_registry",
	"iso_url",
	"nfs_share",
	"ports",
	"registry_mirror",
}

type SchemaOverride struct {
	Description      string
	Default          string
	Type             SchemaType
	DefaultFunc      string
	StateFunc        string
	ValidateDiagFunc string
}

var updateFields = []string{
	"addons",
}

var schemaOverrides map[string]SchemaOverride = map[string]SchemaOverride{
	"memory": {
		Default:          "4g",
		Description:      "Amount of RAM to allocate to Kubernetes (format: <number>[<unit>], where unit = b, k, m or g). Use \"max\" to use the maximum amount of memory. Use \"no-limit\" to not specify a limit (Docker/Podman only))",
		Type:             String,
	},
	"disk_size": {
		Default:          "20000mb",
		Description:      "Disk size allocated to the minikube VM (format: <number>[<unit>(case-insensitive)], where unit = b, k, kb, m, mb, g or gb)",
		Type:             String,
		StateFunc:        "state_utils.ResourceSizeConverter()",
		ValidateDiagFunc: "state_utils.ResourceSizeValidator()",
	},
	"cpus": {
		Default:     "2",
		Description: "Number of CPUs allocated to Kubernetes. Use \"max\" to use the maximum number of CPUs. Use \"no-limit\" to not specify a limit (Docker/Podman only)",
		Type:        String,
	},
	// Customize the description to be the fullset of drivers
	"driver": {
		Default:     "docker",
		Description: "Driver is one of the following - Windows: (hyperv, docker, virtualbox, vmware, qemu2, ssh) - OSX: (virtualbox, parallels, vmwarefusion, hyperkit, vmware, qemu2, docker, podman, ssh) - Linux: (docker, kvm2, virtualbox, qemu2, none, podman, ssh)",
		Type:        String,
	},
	"container_runtime": {
		Default:     "docker",
		Description: "The container runtime to be used. Valid options: docker, cri-o, containerd (default: docker)",
		Type:        String,
	},
	// Default schema to unix file paths first and let the provider translate them during runtime
	"mount_string": {
		Description: "The argument to pass the minikube mount command on start.",
		Type:        String,
		DefaultFunc: `func() (any, error) {
				if runtime.GOOS == "windows" {
					home, err := os.UserHomeDir()
					if err != nil {
						return nil, err
					}
					return home + ":" + "/minikube-host", nil
				} else if runtime.GOOS == "darwin" {
					return "/Users:/minikube-host", nil
				} 
				return "/home:/minikube-host", nil
			}`,
	},
	"extra_config": {
		Description: "A set of key=value pairs that describe configuration that may be passed to different components. 		The key should be '.' separated, and the first part before the dot is the component to apply the configuration to. 		Valid components are: kubelet, kubeadm, apiserver, controller-manager, etcd, proxy, scheduler 		Valid kubeadm parameters: ignore-preflight-errors, dry-run, kubeconfig, kubeconfig-dir, node-name, cri-socket, experimental-upload-certs, certificate-key, rootfs, skip-phases, pod-network-cidr",
		Type:        Array,
	},
	"socket_vmnet_path": {
		Description: "Path to socket vmnet binary (QEMU driver only)",
		Type:        String,
		DefaultFunc: `func() (any, error) {
        var prefix string
        if runtime.GOARCH == "arm64" {
            prefix = "/opt/homebrew"
        } else {
            prefix = "/usr/local"
        }
        return prefix + "/var/run/socket_vmnet", nil
    }`,
	},
	"socket_vmnet_client_path": {
		Description: "Path to the socket vmnet client binary (QEMU driver only)",
		Type:        String,
		DefaultFunc: `func() (any, error) {
        var prefix string
        if runtime.GOARCH == "arm64" {
            prefix = "/opt/homebrew"
        } else {
            prefix = "/usr/local"
        }
        return prefix + "/opt/socket_vmnet/bin/socket_vmnet_client", nil
    }`,
	},
}

func run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "minikube", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

type SchemaEntry struct {
	Parameter        string
	Default          string
	DefaultFunc      string
	StateFunc        string
	ValidateDiagFunc string
	Description      string
	Type             SchemaType
	ArrayType        SchemaType
}

type SchemaBuilder struct {
	targetFile string
	minikube   MinikubeBinary
}

type SchemaType string

const (
	String SchemaType = "String"
	Int    SchemaType = "Int"
	Bool   SchemaType = "Bool"
	Array  SchemaType = "Set"
)

func NewSchemaBuilder(targetFile string, minikube MinikubeBinary) *SchemaBuilder {
	return &SchemaBuilder{targetFile: targetFile, minikube: minikube}
}

func (s *SchemaBuilder) Build() (string, error) {
	minikubeVersion, err := s.minikube.GetVersion(context.Background())
	if err != nil {
		return "", errors.New("could not run minikube binary. please ensure that you have minikube installed and available")
	}

	log.Printf("building schema for minikube version: %s", minikubeVersion)

	help, err := s.minikube.GetStartHelpText(context.Background())
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(help))

	entries := make([]SchemaEntry, 0)

	currentEntry := SchemaEntry{}

	pattern := "^-[a-zA-Z], "

	srg := regexp.MustCompile(pattern)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// trim the short parameter e.g. -g
		line = srg.ReplaceAllString(line, "")

		if strings.HasPrefix(line, "--") {
			currentEntry = loadParameter(line)
		} else if line != "" {
			if currentEntry.Description != "" { // Let's read a space between line blocks
				currentEntry.Description += " "
			}
			currentEntry.Description += line
			currentEntry.Description = strings.ReplaceAll(currentEntry.Description, "\\", "\\\\")
			currentEntry.Description = strings.ReplaceAll(currentEntry.Description, "\"", "\\\"")
		} else if currentEntry.Parameter != "" {
			// Apply description override once we've built the description
			val, ok := schemaOverrides[currentEntry.Parameter]
			if ok {
				currentEntry.Description = val.Description
			}

			entries, err = addEntry(entries, currentEntry)
			if err != nil {
				return "", err
			}

			currentEntry.Parameter = ""
		}
	}

	schema := constructSchema(entries)

	return schema, err
}

func loadParameter(line string) SchemaEntry {
	schemaEntry := SchemaEntry{}
	schemaEntry.Description = ""
	seg := strings.Split(line, "=")
	schemaEntry.Parameter = strings.TrimPrefix(seg[0], "--")
	schemaEntry.Parameter = strings.Replace(schemaEntry.Parameter, "-", "_", -1)
	schemaEntry.Default = strings.TrimSuffix(seg[1], ":")
	schemaEntry.Default = strings.ReplaceAll(schemaEntry.Default, "\\", "\\\\")
	schemaEntry.Type = getSchemaType(schemaEntry.Default)

	// Apply explicit overrides
	val, ok := schemaOverrides[schemaEntry.Parameter]
	if ok {
		schemaEntry.Default = val.Default
		schemaEntry.DefaultFunc = val.DefaultFunc
		schemaEntry.Type = val.Type
		schemaEntry.StateFunc = val.StateFunc
		schemaEntry.ValidateDiagFunc = val.ValidateDiagFunc
	}

	if schemaEntry.Type == String {
		schemaEntry.Default = strings.Trim(schemaEntry.Default, "'")
	}

	return schemaEntry
}

func addEntry(entries []SchemaEntry, currentEntry SchemaEntry) ([]SchemaEntry, error) {
	switch currentEntry.Type {
	case String:
		entries = append(entries, SchemaEntry{
			Parameter:        currentEntry.Parameter,
			Default:          fmt.Sprintf("\"%s\"", currentEntry.Default),
			Type:             currentEntry.Type,
			Description:      currentEntry.Description,
			DefaultFunc:      currentEntry.DefaultFunc,
			StateFunc:        currentEntry.StateFunc,
			ValidateDiagFunc: currentEntry.ValidateDiagFunc,
		})
	case Bool:
		entries = append(entries, SchemaEntry{
			Parameter:        currentEntry.Parameter,
			Default:          currentEntry.Default,
			Type:             currentEntry.Type,
			Description:      currentEntry.Description,
			DefaultFunc:      currentEntry.DefaultFunc,
			StateFunc:        currentEntry.StateFunc,
			ValidateDiagFunc: currentEntry.ValidateDiagFunc,
		})
	case Int:
		val, err := strconv.Atoi(currentEntry.Default)
		if err != nil {
			// is it a timestamp?
			time, err := time.ParseDuration(currentEntry.Default)
			if err != nil {
				return nil, err
			}
			val = int(time.Minutes())
			currentEntry.Description = fmt.Sprintf("%s (Configured in minutes)", currentEntry.Description)
		}
		entries = append(entries, SchemaEntry{
			Parameter:        currentEntry.Parameter,
			Default:          strconv.Itoa(val),
			Type:             currentEntry.Type,
			Description:      currentEntry.Description,
			DefaultFunc:      currentEntry.DefaultFunc,
			StateFunc:        currentEntry.StateFunc,
			ValidateDiagFunc: currentEntry.ValidateDiagFunc,
		})
	case Array:
		entries = append(entries, SchemaEntry{
			Parameter:        currentEntry.Parameter,
			Type:             Array,
			ArrayType:        String,
			Description:      currentEntry.Description,
			DefaultFunc:      currentEntry.DefaultFunc,
			StateFunc:        currentEntry.StateFunc,
			ValidateDiagFunc: currentEntry.ValidateDiagFunc,
		})
	}

	return entries, nil
}

func (s *SchemaBuilder) Write(schema string) error {
	return os.WriteFile(s.targetFile, []byte(schema), 0644)
}

func constructSchema(entries []SchemaEntry) string {

	header := `//go:generate go run ../generate/main.go -target $GOFILE
// THIS FILE IS GENERATED DO NOT EDIT
package minikube

import (
	"runtime"
	"os"

	"github.com/scott-the-programmer/terraform-provider-minikube/minikube/state_utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	clusterSchema = map[string]*schema.Schema{
		"cluster_name": {
			Type:					schema.TypeString,
			Optional:			true,
			ForceNew:			true,
			Description:	"The name of the minikube cluster",
			Default:			"terraform-provider-minikube",
		},

		"client_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "client key for cluster",
			Sensitive:   true,
		},

		"client_certificate": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "client certificate used in cluster",
			Sensitive:   true,
		},

		"cluster_ca_certificate": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "certificate authority for cluster",
			Sensitive:   true,
		},

		"host": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "the host name for the cluster",
		},
`

	body := ""
	for _, entry := range entries {
		extraParams := ""
		if contains(computedFields, entry.Parameter) {
			extraParams = `
			Computed:			true,
			`
		}

		if !contains(updateFields, entry.Parameter) {
			extraParams += `
			Optional:			true,
			ForceNew:			true,
			`
		} else {
			extraParams += `
			Optional:			true,
			`
		}

		if entry.Type == Array {
			extraParams += fmt.Sprintf(`
			Elem: &schema.Schema{
				Type:	%s,
			},
			`, "schema.Type"+entry.ArrayType)
		} else if entry.DefaultFunc != "" {
			extraParams += fmt.Sprintf(`
			DefaultFunc:	%s,`, entry.DefaultFunc)
		} else {
			extraParams += fmt.Sprintf(`
			Default:	%s,`, entry.Default)
		}

		if entry.StateFunc != "" {
			extraParams += fmt.Sprintf(`
			StateFunc:	%s,`, entry.StateFunc)
		}

		if entry.ValidateDiagFunc != "" {
			extraParams += fmt.Sprintf(`
			ValidateDiagFunc:	%s,`, entry.ValidateDiagFunc)
		}

		body = body + fmt.Sprintf(`
		"%s": {
			Type:					%s,
			Description:	"%s",
			%s
		},
	`, entry.Parameter, "schema.Type"+entry.Type, entry.Description, extraParams)
	}

	footer := `
	}
)

func GetClusterSchema() map[string]*schema.Schema {
	return clusterSchema
}
	`

	return header + body + footer
}

func getSchemaType(s string) SchemaType {
	if strings.Count(s, "'") == 2 || s == "" {
		return String
	} else if s == "true" || s == "false" {
		return Bool
	} else if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return Array
	}
	return Int
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
