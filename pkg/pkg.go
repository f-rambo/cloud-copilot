package pkg

import (
	"github.com/f-rambo/ocean/pkg/ansible"
	"github.com/f-rambo/ocean/pkg/argoworkflows"
	"github.com/f-rambo/ocean/pkg/githubapi"
	"github.com/f-rambo/ocean/pkg/helm"
	"github.com/f-rambo/ocean/pkg/kubernetes"
	"github.com/f-rambo/ocean/pkg/pulumi"
	"github.com/f-rambo/ocean/pkg/sailor"
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
