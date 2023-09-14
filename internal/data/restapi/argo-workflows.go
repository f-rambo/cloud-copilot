package restapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	argoJson "github.com/argoproj/pkg/json"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/pkg/errors"
)

type ArgoWorkflows struct {
	argo apiclient.Client
}

func NewArgoWorkflows(c conf.ArgoWorkflows) (*ArgoWorkflows, error) {
	argoServer := apiclient.ArgoServerOpts{
		URL: fmt.Sprintf("%s:%d", c.GetHost(), c.GetPort()),
	}
	_, client, err := apiclient.NewClientFromOpts(apiclient.Opts{
		ArgoServerOpts: argoServer,
		AuthSupplier: func() string {
			return c.GetToken()
		},
	})
	if err != nil {
		return nil, err
	}
	return &ArgoWorkflows{argo: client}, nil
}

func (a *ArgoWorkflows) GetWorkflows(ctx context.Context, namespace string) (*wfv1.WorkflowList, error) {
	workflowServiceClient := a.argo.NewWorkflowServiceClient()
	return workflowServiceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: namespace})
}

func (a *ArgoWorkflows) GetWorkflow(ctx context.Context, namespace, name string) (*wfv1.Workflow, error) {
	workflowServiceClient := a.argo.NewWorkflowServiceClient()
	return workflowServiceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{Namespace: namespace, Name: name})
}

func (a *ArgoWorkflows) DeleteWorkflow(ctx context.Context, namespace, name string) error {
	workflowServiceClient := a.argo.NewWorkflowServiceClient()
	_, err := workflowServiceClient.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Namespace: namespace, Name: name})
	return err
}

// 1. 提交模版、提交wf
func (a *ArgoWorkflows) CreateWorkflows(ctx context.Context, workflows string) error {
	wfs, err := a.unmarshalWorkflows([]byte(workflows), true)
	if err != nil {
		return err
	}
	workflowServiceClient := a.argo.NewWorkflowServiceClient()
	for _, wf := range wfs {
		_, err = workflowServiceClient.CreateWorkflow(ctx, &workflowpkg.WorkflowCreateRequest{
			Namespace: wf.Namespace,
			Workflow:  &wf,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *ArgoWorkflows) SubmitWorkflow(ctx context.Context, resourceIdentifier, namespace string) error {
	// kind ref :https://pkg.go.dev/github.com/argoproj/argo-workflows/v3@v3.4.11/pkg/apis/workflow
	parts := strings.SplitN(resourceIdentifier, "/", 2)
	if len(parts) != 2 {
		return errors.New("resource identifier is malformed. Should be `kind/name`, e.g. cronwf/hello-world-cwf")
	}
	kind := parts[0]
	name := parts[1]

	workflowServiceClient := a.argo.NewWorkflowServiceClient()
	_, err := workflowServiceClient.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
		Namespace:    namespace,
		ResourceKind: kind,
		ResourceName: name,
	})
	return err
}

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func (a *ArgoWorkflows) unmarshalWorkflows(wfBytes []byte, strict bool) ([]wfv1.Workflow, error) {
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}, nil
	}
	yamlWfs, err := wfcommon.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}
	return nil, err
}
