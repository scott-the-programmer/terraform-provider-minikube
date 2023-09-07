package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"

	mlog "github.com/docker/machine/libmachine/log"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/klog/v2"
)

var (
	machineLogErrorRe       = regexp.MustCompile(`VirtualizationException`)
	machineLogWarningRe     = regexp.MustCompile(`(?i)warning`)
	machineLogEnvironmentRe = regexp.MustCompile(`&exec\.Cmd`)
)

type machineLogBridge struct{}

// Write passes machine driver logs to klog
func (lb machineLogBridge) Write(b []byte) (n int, err error) {
	if machineLogEnvironmentRe.Match(b) {
		return len(b), nil
	} else if machineLogErrorRe.Match(b) {
		tflog.Error(context.TODO(), fmt.Sprintf("libmachine: %s", b))
	} else if machineLogWarningRe.Match(b) {
		tflog.Warn(context.TODO(), fmt.Sprintf("libmachine: %s", b))
	} else {
		tflog.Info(context.TODO(), fmt.Sprintf("libmachine: %s", b))
	}
	return len(b), nil
}

type stdLogBridge struct{}

func (lb stdLogBridge) Write(b []byte) (n int, err error) {
	// Split "d.go:23: message" into "d.go", "23", and "message".
	parts := bytes.SplitN(b, []byte{':'}, 3)
	if len(parts) != 3 || len(parts[0]) < 1 || len(parts[2]) < 1 {
		klog.Errorf("bad log format: %s", b)
		return
	}

	file := string(parts[0])
	text := string(parts[2][1:]) // skip leading space
	line, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		text = fmt.Sprintf("bad line number: %s", b)
		line = 0
	}
	tflog.Info(context.TODO(), fmt.Sprintf("stdlog: %s:%d %s", file, line, text))
	return len(b), nil
}

type tfLogBridge struct{}

func (lb tfLogBridge) Write(b []byte) (n int, err error) {
	tflog.Info(context.TODO(), string(b))
	return len(b), nil
}

func registerLogging() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(stdLogBridge{})
	mlog.SetErrWriter(machineLogBridge{})
	mlog.SetOutWriter(machineLogBridge{})
	klog.SetOutput(tfLogBridge{})
	mlog.SetDebug(false)
}
