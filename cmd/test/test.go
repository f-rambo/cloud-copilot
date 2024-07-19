package test

import (
	"context"
	"fmt"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/interfaces"
	"github.com/f-rambo/ocean/internal/server"
	"github.com/f-rambo/ocean/mocks"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang/mock/gomock"
)

func newContext() context.Context {
	return context.Background()
}

func newApp(ctx context.Context, logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID("test"),
		kratos.Name("test"),
		kratos.Version("v1.0.0"),
		kratos.Context(ctx),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)
}

func wireApp(controller *gomock.Controller, bootstrap *conf.Bootstrap, logger log.Logger) (*kratos.App, func(), error) {
	context := newContext()
	// cluster
	clusterRepo := clusterRepo(controller)
	infrastructure := clusterInfrastructure(controller)
	clusterConstruct := clusterConstruct(controller)
	clusterRuntime := clusterRuntime(controller)
	// project
	projectRepo := projectRepo(controller)
	clusterPorjectRepo := projectClusterRepo(controller)
	// app
	appRepo := appRepo(controller)
	sailorRepo := sailorRepo(controller)
	appRuntime := appRuntime(controller)
	appConstruct := appConstruct(controller)
	// user
	userRepo := userRepo(controller)
	thirdparty := githubClient(controller)
	// services
	servicesRepo := servicesRepo(controller)
	workflowRepo := workflowRepo(controller)
	// biz
	clusterUsecase := biz.NewClusterUseCase(clusterRepo, infrastructure, clusterConstruct, clusterRuntime, logger)
	servicesUseCase := biz.NewServicesUseCase(servicesRepo, workflowRepo, logger)
	userUseCase := biz.NewUseUser(userRepo, thirdparty, logger)
	appUsecase := biz.NewAppUsecase(appRepo, clusterRepo, projectRepo, sailorRepo, appRuntime, appConstruct, logger, bootstrap)
	projectUsecase := biz.NewProjectUseCase(projectRepo, clusterPorjectRepo, logger)
	servicesInterface := interfaces.NewServicesInterface(servicesUseCase, projectUsecase)
	userInterface := interfaces.NewUserInterface(userUseCase, bootstrap)
	projectInterface := interfaces.NewProjectInterface(projectUsecase, appUsecase, clusterUsecase, bootstrap, logger)
	clusterInterface := interfaces.NewClusterInterface(clusterUsecase, projectUsecase, appUsecase, bootstrap, logger)
	appInterface := interfaces.NewAppInterface(appUsecase, userUseCase, bootstrap, logger)
	autoscaler := interfaces.NewAutoscaler(clusterUsecase, bootstrap, logger)
	grpcServer := server.NewGRPCServer(bootstrap, clusterInterface, appInterface, servicesInterface, userInterface, projectInterface, autoscaler, logger)
	httpServer := server.NewHTTPServer(bootstrap, clusterInterface, appInterface, servicesInterface, userInterface, projectInterface, logger)
	app := newApp(context, logger, grpcServer, httpServer)
	return app, func() {
	}, nil
}

// cluster

var nodeGroup *biz.NodeGroup = &biz.NodeGroup{
	ID:                      1,
	MinSize:                 3,
	MaxSize:                 6,
	Type:                    "testType",
	InstanceType:            "testInstanceType",
	OSImage:                 "testOSImage",
	CPU:                     1,
	Memory:                  1.0,
	GPU:                     1,
	GpuSpec:                 "testGpuSpec",
	SystemDisk:              1,
	DataDisk:                1,
	InternetMaxBandwidthOut: 1,
	NodeInitScript:          "testNodeInitScript",
	TargetSize:              3,
	ClusterID:               1,
}

var node *biz.Node = &biz.Node{
	ID:          1,
	NodeGroup:   nodeGroup,
	Name:        "minikube",
	Status:      biz.NodeStatusRunning.Uint8(),
	Role:        "master",
	InternalIP:  "127.0.0.1",
	ExternalIP:  "127.0.0.1",
	User:        "testUser",
	Labels:      "testLabels",
	ErrorInfo:   "testErrorInfo",
	ClusterID:   1,
	NodeGroupID: 1,
	NodePrice:   1.0,
	PodPrice:    1.0,
}

var cluster *biz.Cluster = &biz.Cluster{
	ID:         1,
	Name:       "testCluster",
	Nodes:      []*biz.Node{node},
	NodeGroups: []*biz.NodeGroup{nodeGroup},
	Status:     biz.ClusterStatusRunning.Uint8(),
	Type:       "testType",
	Region:     "testRegion",
	VpcID:      "testVpcID",
	ExternalIP: "testExternalIP",
	AccessID:   "testAccessID",
	AccessKey:  "testAccessKey",
	GPULabel:   "testGPULabel",
	GPUTypes:   "testGPUTypes",
}

func clusterRepo(controller *gomock.Controller) biz.ClusterRepo {
	mc := mocks.NewMockClusterRepo(controller)
	mc.EXPECT().GetByName(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, name string) (*biz.Cluster, error) {
		if name != "testCluster" {
			return nil, fmt.Errorf("cluster not found")
		}
		return cluster, nil
	})
	mc.EXPECT().Save(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, paramCluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		if cluster.Name != "testCluster" {
			return fmt.Errorf("cluster not found")
		}
		cluster = paramCluster
		return nil
	})
	return mc
}

func clusterInfrastructure(controller *gomock.Controller) biz.Infrastructure {
	mc := mocks.NewMockInfrastructure(controller)
	mc.EXPECT().DeleteServers(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().SaveServers(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	return mc
}

func clusterConstruct(controller *gomock.Controller) biz.ClusterConstruct {
	mc := mocks.NewMockClusterConstruct(controller)
	mc.EXPECT().GenerateInitialCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().UnInstallCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().MigrateToBostionHost(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().InstallCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().AddNodes(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	mc.EXPECT().RemoveNodes(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
		if cluster == nil {
			return fmt.Errorf("cluster is nil")
		}
		return nil
	})
	return mc
}

func clusterRuntime(controller *gomock.Controller) biz.ClusterRuntime {
	mc := mocks.NewMockClusterRuntime(controller)
	mc.EXPECT().CurrentCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context) (*biz.Cluster, error) {
		return cluster, nil
	})
	return mc
}

// project
func projectRepo(controller *gomock.Controller) biz.ProjectRepo {
	return mocks.NewMockProjectRepo(controller)
}

func projectClusterRepo(controller *gomock.Controller) biz.ClusterPorjectRepo {
	return mocks.NewMockClusterPorjectRepo(controller)
}

// app
func appRepo(controller *gomock.Controller) biz.AppRepo {
	m := mocks.NewMockAppRepo(controller)
	m.EXPECT().CreateAppType(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, appType *biz.AppType) error {
		if appType.Name == "test" {
			return fmt.Errorf("app type already exists")
		}
		return nil
	})
	return m
}

func sailorRepo(controller *gomock.Controller) biz.SailorRepo {
	return mocks.NewMockSailorRepo(controller)
}

func appRuntime(controller *gomock.Controller) biz.AppRuntime {
	return mocks.NewMockAppRuntime(controller)
}

func appConstruct(controller *gomock.Controller) biz.AppConstruct {
	return mocks.NewMockAppConstruct(controller)
}

// user
func userRepo(controller *gomock.Controller) biz.UserRepo {
	return mocks.NewMockUserRepo(controller)
}

func githubClient(controller *gomock.Controller) biz.Thirdparty {
	return mocks.NewMockThirdparty(controller)
}

// services
func servicesRepo(controller *gomock.Controller) biz.ServicesRepo {
	return mocks.NewMockServicesRepo(controller)
}

func workflowRepo(controller *gomock.Controller) biz.WorkflowRepo {
	return mocks.NewMockWorkflowRepo(controller)
}
