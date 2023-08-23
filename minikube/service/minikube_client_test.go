package service

import (
	"errors"
	"reflect"
	"sort"
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"k8s.io/minikube/pkg/minikube/config"
	_ "k8s.io/minikube/pkg/minikube/registry/drvs"
)

func TestMinikubeClient_Start(t *testing.T) {
	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nRunner         Cluster
		dLoader         Downloader
		nodes           int
		tfCreationLock  sync.Mutex
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getNodeSuccess(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: false,
		},
		{
			name: "Success With Addons",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons: []string{
					"mock_addon",
				},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getNodeSuccess(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: false,
		},
		{
			name: "Success With Nodes",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons: []string{
					"mock_addon",
				},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getMultipleNodesSuccess(ctrl, 3),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           3,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: false,
		},
		{
			name: "Failure On Adding Nodes",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons: []string{
					"mock_addon",
				},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getMultipleNodesFailure(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           3,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: true,
		},
		{
			name: "Download Failure",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         nil,
				dLoader:         getDownloadFailure(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: true,
		},
		{
			name: "Tarball Failure",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         nil,
				dLoader:         getTarballFailure(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: true,
		},
		{
			name: "Provision Failure",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getProvisionerFailure(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: true,
		},
		{
			name: "Start Failure",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getStartFailure(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
				nodes:           1,
				tfCreationLock:  sync.Mutex{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nRunner:         tt.fields.nRunner,
				dLoader:         tt.fields.dLoader,
				nodes:           tt.fields.nodes,
			}
			if _, err := e.Start(); (err != nil) != tt.wantErr {
				t.Errorf("MinikubeClient.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinikubeClient_Delete(t *testing.T) {
	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nRunner         Cluster
		dLoader         Downloader
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getDeleteSuccess(ctrl),
				dLoader:         &MockDownloader{},
			},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				clusterConfig: config.ClusterConfig{
					Nodes: []config.Node{
						{},
					},
				},
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: true,
				nRunner:         getDeleteFailure(ctrl),
				dLoader:         &MockDownloader{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nRunner:         tt.fields.nRunner,
				dLoader:         tt.fields.dLoader,
			}
			if err := e.Delete(); (err != nil) != tt.wantErr {
				t.Errorf("MinikubeClient.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMinikubeClient(t *testing.T) {
	type args struct {
		args MinikubeClientConfig
		dep  MinikubeClientDeps
	}
	tests := []struct {
		name string
		args args
		want *MinikubeClient
	}{
		{
			name: "Blank Ctor",

			args: args{
				args: MinikubeClientConfig{},
				dep:  MinikubeClientDeps{},
			},
			want: &MinikubeClient{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMinikubeClient(tt.args.args, tt.args.dep); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMinikubeClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinikubeClient_SetConfig(t *testing.T) {
	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nodes           int
		TfCreationLock  *sync.Mutex
		K8sVersion      string
		nRunner         Cluster
		dLoader         Downloader
	}
	type args struct {
		args MinikubeClientConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "Sets Cluster Properties",
			fields: fields{},
			args: args{
				args: MinikubeClientConfig{
					ClusterName: "mock",
					Nodes:       100,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nodes:           tt.fields.nodes,
				TfCreationLock:  tt.fields.TfCreationLock,
				K8sVersion:      tt.fields.K8sVersion,
				nRunner:         tt.fields.nRunner,
				dLoader:         tt.fields.dLoader,
			}
			e.SetConfig(tt.args.args)

			if e.clusterName != tt.args.args.ClusterName {
				t.Errorf("cluster name = %v, want %v", e.clusterConfig, tt.args.args.ClusterName)
			}
		})
	}
}

func TestMinikubeClient_SetDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)

	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nodes           int
		TfCreationLock  *sync.Mutex
		K8sVersion      string
		nRunner         Cluster
		dLoader         Downloader
	}
	type args struct {
		dep MinikubeClientDeps
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "Should Set Dependencies",
			fields: fields{},
			args: args{
				dep: MinikubeClientDeps{
					Node:       NewMockCluster(ctrl),
					Downloader: NewMockDownloader(ctrl),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nodes:           tt.fields.nodes,
				TfCreationLock:  tt.fields.TfCreationLock,
				K8sVersion:      tt.fields.K8sVersion,
				nRunner:         tt.fields.nRunner,
				dLoader:         tt.fields.dLoader,
			}
			e.SetDependencies(tt.args.dep)
		})
	}
}

func TestMinikubeClient_GetConfig(t *testing.T) {
	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nodes           int
		TfCreationLock  *sync.Mutex
		K8sVersion      string
		nRunner         Cluster
		dLoader         Downloader
	}
	tests := []struct {
		name   string
		fields fields
		want   MinikubeClientConfig
	}{
		{
			name: "Retrieves client config",
			fields: fields{
				clusterConfig:   config.ClusterConfig{},
				isoUrls:         []string{"url1", "url2"},
				clusterName:     "abc",
				addons:          []string{"addon1", "addon2"},
				deleteOnFailure: false,
				nodes:           1,
			},
			want: MinikubeClientConfig{
				ClusterConfig:   config.ClusterConfig{},
				IsoUrls:         []string{"url1", "url2"},
				ClusterName:     "abc",
				Addons:          []string{"addon1", "addon2"},
				DeleteOnFailure: false,
				Nodes:           1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nodes:           tt.fields.nodes,
				TfCreationLock:  tt.fields.TfCreationLock,
				K8sVersion:      tt.fields.K8sVersion,
				nRunner:         tt.fields.nRunner,
				dLoader:         tt.fields.dLoader,
			}
			if got := e.GetConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MinikubeClient.GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinikubeClient_ApplyAddons(t *testing.T) {
	type fields struct {
		clusterConfig   config.ClusterConfig
		clusterName     string
		addons          []string
		isoUrls         []string
		deleteOnFailure bool
		nodes           int
		TfCreationLock  *sync.Mutex
		K8sVersion      string
		dLoader         Downloader
	}
	type args struct {
		addons []string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		addAddons    []string
		deleteAddons []string
	}{
		{
			name: "Should remove existing addons",
			fields: fields{
				clusterName:    "cluster",
				addons:         []string{"feature1", "feature2"},
				TfCreationLock: &sync.Mutex{},
			},
			args: args{
				addons: []string{"feature1"},
			},
			wantErr:      false,
			deleteAddons: []string{"feature2"},
		},
		{
			name: "Should add new addons",
			fields: fields{
				clusterName:    "cluster",
				addons:         []string{"feature1", "feature2"},
				TfCreationLock: &sync.Mutex{},
			},
			args: args{
				addons: []string{"feature1", "feature2", "feature3"},
			},
			wantErr:   false,
			addAddons: []string{"feature3"},
		},
		{
			name: "Should remove and add addons",
			fields: fields{
				clusterName:    "cluster",
				addons:         []string{"feature1", "feature2"},
				TfCreationLock: &sync.Mutex{},
			},
			args: args{
				addons: []string{"feature3"},
			},
			wantErr:      false,
			deleteAddons: []string{"feature1", "feature2"},
			addAddons:    []string{"feature3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockNode := NewMockCluster(ctrl)
			delSeq := make([]*gomock.Call, 0)
			addSeq := make([]*gomock.Call, 0)
			for _, deleteAddon := range tt.deleteAddons {
				delSeq = append(delSeq, mockNode.EXPECT().
					SetAddon("cluster", deleteAddon, "false").
					Return(nil))
			}
			for _, addAddon := range tt.addAddons {
				addSeq = append(addSeq, mockNode.EXPECT().
					SetAddon("cluster", addAddon, "true").
					Return(nil))
			}
			gomock.InAnyOrder(append(delSeq, addSeq...))

			e := &MinikubeClient{
				clusterConfig:   tt.fields.clusterConfig,
				clusterName:     tt.fields.clusterName,
				addons:          tt.fields.addons,
				isoUrls:         tt.fields.isoUrls,
				deleteOnFailure: tt.fields.deleteOnFailure,
				nodes:           tt.fields.nodes,
				TfCreationLock:  tt.fields.TfCreationLock,
				K8sVersion:      tt.fields.K8sVersion,
				nRunner:         mockNode,
				dLoader:         tt.fields.dLoader,
			}
			if err := e.ApplyAddons(tt.args.addons); (err != nil) != tt.wantErr {
				t.Errorf("MinikubeClient.EnableAddons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinikubeClient_GetAddons(t *testing.T) {

	type fields struct {
		addons map[string]bool
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Should convert enabled addons into slice",
			fields: fields{
				addons: map[string]bool{
					"addon1": true,
					"addon2": false,
					"addon3": true,
				},
			},
			want: []string{"addon1", "addon3"},
		},
		{
			name: "Returns empty slice",
			fields: fields{
				addons: map[string]bool{
					"addon1": false,
					"addon2": false,
					"addon3": false,
				},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		ctrl := gomock.NewController(t)
		mockCluster := NewMockCluster(ctrl)
		mockCluster.EXPECT().
			Get(gomock.Any()).
			Return(
				&config.ClusterConfig{
					Addons: tt.fields.addons,
				},
			)

		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				nRunner: mockCluster,
			}
			got := e.GetAddons()
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MinikubeClient.GetAddons() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinikubeClient_ContainerMounts(t *testing.T) {

	type fields struct {
		mount       bool
		mountString string
		driver      string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Should set container mounts when provided mount string",
			fields: fields{
				mount:       true,
				mountString: "/test:/data",
				driver:      "docker",
			},
			want: []string{"/test:/data"},
		},
		{
			name: "Should not set container mounts for non container drivers",
			fields: fields{
				mount:       true,
				mountString: "/test:/data",
				driver:      "other",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		ctrl := gomock.NewController(t)

		t.Run(tt.name, func(t *testing.T) {
			e := &MinikubeClient{
				clusterConfig: config.ClusterConfig{
					Driver:      tt.fields.driver,
					Mount:       tt.fields.mount,
					MountString: tt.fields.mountString,
					Nodes: []config.Node{
						{},
					},
				},
				clusterName:     "sut",
				addons:          []string{},
				isoUrls:         []string{},
				deleteOnFailure: false,
				nRunner:         getNodeSuccess(ctrl),
				dLoader:         getDownloadSuccess(ctrl),
			}
			e.Start()
			got := e.clusterConfig.ContainerVolumeMounts
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clusterConfig.ContainerVolumeMounts = %v, want %v", got, tt.want)
			}
		})
	}
}

func getProvisionerFailure(ctrl *gomock.Controller) Cluster {
	nRunnerProvisionFailure := NewMockCluster(ctrl)

	nRunnerProvisionFailure.EXPECT().
		Provision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, false, nil, nil, errors.New("provision error"))

	return nRunnerProvisionFailure
}

func getStartFailure(ctrl *gomock.Controller) Cluster {
	nRunnerStartFailure := NewMockCluster(ctrl)

	nRunnerStartFailure.EXPECT().
		Provision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, false, nil, nil, nil)

	nRunnerStartFailure.EXPECT().
		Start(gomock.Any(), true).
		Return(nil, errors.New("start error"))

	return nRunnerStartFailure
}

func getDownloadFailure(ctrl *gomock.Controller) Downloader {
	dLoaderFailure := NewMockDownloader(ctrl)

	dLoaderFailure.EXPECT().
		ISO(gomock.Any(), gomock.Any()).
		Return("", errors.New("download error"))

	return dLoaderFailure
}

func getTarballFailure(ctrl *gomock.Controller) Downloader {
	dLoaderSuccess := NewMockDownloader(ctrl)

	dLoaderSuccess.EXPECT().
		ISO(gomock.Any(), gomock.Any()).
		Return("https://mock_iso_url/iso.iso", nil)

	dLoaderSuccess.EXPECT().
		PreloadTarball(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("tar ball failure"))

	return dLoaderSuccess
}

func getNodeSuccess(ctrl *gomock.Controller) Cluster {
	nRunnerSuccess := NewMockCluster(ctrl)

	nRunnerSuccess.EXPECT().
		Provision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, false, nil, nil, nil)

	nRunnerSuccess.EXPECT().
		Start(gomock.Any(), true).
		Return(nil, nil)

	return nRunnerSuccess
}

func getMultipleNodesSuccess(ctrl *gomock.Controller, n int) Cluster {
	nRunnerSuccess := NewMockCluster(ctrl)

	nRunnerSuccess.EXPECT().
		Provision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, false, nil, nil, nil)

	nRunnerSuccess.EXPECT().
		Start(gomock.Any(), true).
		Return(nil, nil)

	nRunnerSuccess.EXPECT().
		Add(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(n - 1)

	return nRunnerSuccess
}

func getMultipleNodesFailure(ctrl *gomock.Controller) Cluster {
	nRunnerSuccess := NewMockCluster(ctrl)

	nRunnerSuccess.EXPECT().
		Provision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, false, nil, nil, nil)

	nRunnerSuccess.EXPECT().
		Start(gomock.Any(), true).
		Return(nil, nil)

	nRunnerSuccess.EXPECT().
		Add(gomock.Any(), gomock.Any()).
		Return(errors.New("error adding node"))

	return nRunnerSuccess
}

func getDownloadSuccess(ctrl *gomock.Controller) Downloader {
	dLoaderSuccess := NewMockDownloader(ctrl)

	dLoaderSuccess.EXPECT().
		ISO(gomock.Any(), gomock.Any()).
		Return("https://mock_iso_url/iso.iso", nil)

	dLoaderSuccess.EXPECT().
		PreloadTarball(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	return dLoaderSuccess
}

func getDeleteSuccess(ctrl *gomock.Controller) Cluster {
	nRunnerSuccess := NewMockCluster(ctrl)

	nRunnerSuccess.EXPECT().
		Delete(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	return nRunnerSuccess
}

func getDeleteFailure(ctrl *gomock.Controller) Cluster {
	nRunnerSuccess := NewMockCluster(ctrl)

	nRunnerSuccess.EXPECT().
		Delete(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("delete error"))

	return nRunnerSuccess
}
