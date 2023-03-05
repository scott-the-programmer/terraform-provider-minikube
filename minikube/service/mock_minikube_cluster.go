// Code generated by MockGen. DO NOT EDIT.
// Source: minikube_cluster.go

// Package service is a generated GoMock package.
package service

import (
	reflect "reflect"

	libmachine "github.com/docker/machine/libmachine"
	host "github.com/docker/machine/libmachine/host"
	gomock "github.com/golang/mock/gomock"
	command "k8s.io/minikube/pkg/minikube/command"
	config "k8s.io/minikube/pkg/minikube/config"
	kubeconfig "k8s.io/minikube/pkg/minikube/kubeconfig"
	mustload "k8s.io/minikube/pkg/minikube/mustload"
	node "k8s.io/minikube/pkg/minikube/node"
)

// MockCluster is a mock of Cluster interface.
type MockCluster struct {
	ctrl     *gomock.Controller
	recorder *MockClusterMockRecorder
}

// MockClusterMockRecorder is the mock recorder for MockCluster.
type MockClusterMockRecorder struct {
	mock *MockCluster
}

// NewMockCluster creates a new mock instance.
func NewMockCluster(ctrl *gomock.Controller) *MockCluster {
	mock := &MockCluster{ctrl: ctrl}
	mock.recorder = &MockClusterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCluster) EXPECT() *MockClusterMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockCluster) Add(cc *config.ClusterConfig, starter node.Starter) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", cc, starter)
	ret0, _ := ret[0].(error)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockClusterMockRecorder) Add(cc, starter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockCluster)(nil).Add), cc, starter)
}

// Delete mocks base method.
func (m *MockCluster) Delete(cc config.ClusterConfig, name string) (*config.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", cc, name)
	ret0, _ := ret[0].(*config.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockClusterMockRecorder) Delete(cc, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockCluster)(nil).Delete), cc, name)
}

// Get mocks base method.
func (m *MockCluster) Get(name string) mustload.ClusterController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", name)
	ret0, _ := ret[0].(mustload.ClusterController)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockClusterMockRecorder) Get(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCluster)(nil).Get), name)
}

// Provision mocks base method.
func (m *MockCluster) Provision(cc *config.ClusterConfig, n *config.Node, apiServer, delOnFail bool) (command.Runner, bool, libmachine.API, *host.Host, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Provision", cc, n, apiServer, delOnFail)
	ret0, _ := ret[0].(command.Runner)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(libmachine.API)
	ret3, _ := ret[3].(*host.Host)
	ret4, _ := ret[4].(error)
	return ret0, ret1, ret2, ret3, ret4
}

// Provision indicates an expected call of Provision.
func (mr *MockClusterMockRecorder) Provision(cc, n, apiServer, delOnFail interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Provision", reflect.TypeOf((*MockCluster)(nil).Provision), cc, n, apiServer, delOnFail)
}

// SetAddon mocks base method.
func (m *MockCluster) SetAddon(name, addon, value string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetAddon", name, addon, value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetAddon indicates an expected call of SetAddon.
func (mr *MockClusterMockRecorder) SetAddon(name, addon, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAddon", reflect.TypeOf((*MockCluster)(nil).SetAddon), name, addon, value)
}

// Start mocks base method.
func (m *MockCluster) Start(starter node.Starter, apiServer bool) (*kubeconfig.Settings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", starter, apiServer)
	ret0, _ := ret[0].(*kubeconfig.Settings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Start indicates an expected call of Start.
func (mr *MockClusterMockRecorder) Start(starter, apiServer interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockCluster)(nil).Start), starter, apiServer)
}