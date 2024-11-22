package repository

import (
	"github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime"
	"github.com/f-rambo/cloud-copilot/internal/repository/infrastructure"
	"github.com/f-rambo/cloud-copilot/internal/repository/sidecar"
	"github.com/google/wire"
)

// ProviderSet is repository providers.
var ProviderSet = wire.NewSet(
	sidecar.NewSidecarCluster,
	infrastructure.NewInfrastructureCluster,
	clusterruntime.NewClusterRuntimeApp,
	clusterruntime.NewClusterRuntimeUser,
	clusterruntime.NewClusterRuntimeProject,
	clusterruntime.NewClusterRuntimeCluster,
	clusterruntime.NewClusterRuntimeService,
)
