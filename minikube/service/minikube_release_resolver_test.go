package service

import (
	"fmt"
	"runtime"
	"testing"
)

func TestGetMinikubeIso(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Retrieves minikube iso",
			want: fmt.Sprintf("https://github.com/kubernetes/minikube/releases/download/v1.31.0/minikube-v1.31.0-%s.iso", runtime.GOARCH),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMinikubeIso(); got != tt.want {
				t.Errorf("GetMinikubeIso() = %v, want %v", got, tt.want)
			}
		})
	}
}
