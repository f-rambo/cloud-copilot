package third_package

import (
	"github.com/f-rambo/ocean/third_package/ansible"
	"github.com/f-rambo/ocean/third_package/argoworkflows"
	"github.com/f-rambo/ocean/third_package/githubapi"
	"github.com/f-rambo/ocean/third_package/helm"
	"github.com/f-rambo/ocean/third_package/kubernetes"
	"github.com/f-rambo/ocean/third_package/pulumi"
	"github.com/f-rambo/ocean/third_package/sailor"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	ansible.NewClusterConstruct,
	argoworkflows.NewWorkflowRepo,
	helm.NewAppConstructRepo,
	kubernetes.NewAppDeployedResource,
	kubernetes.NewClusterRuntime,
	kubernetes.NewProjectClient,
	pulumi.NewClusterInfrastructure,
	sailor.NewSailorClient,
	githubapi.NewUserClient,
)
