package third_package

import (
	"github.com/f-rambo/ocean/third_package/argoworkflows"
	"github.com/f-rambo/ocean/third_package/githubapi"
	"github.com/f-rambo/ocean/third_package/helm"
	infrastructure "github.com/f-rambo/ocean/third_package/infrastructure"
	"github.com/f-rambo/ocean/third_package/kubernetes"
	"github.com/f-rambo/ocean/third_package/sailor"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	argoworkflows.NewWorkflowRepo,
	helm.NewAppConstructRepo,
	kubernetes.NewAppDeployedResource,
	kubernetes.NewClusterRuntime,
	kubernetes.NewProjectClient,
	sailor.NewSailorClient,
	githubapi.NewUserClient,
	infrastructure.NewClusterInfrastructure,
)
