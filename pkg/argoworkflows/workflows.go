package argoworkflows

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/f-rambo/ocean/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	resourceName = "workflows"
)

type WorkflowInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*wfv1.Workflow, error)
	Create(context.Context, *wfv1.Workflow) (*wfv1.Workflow, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	// ...
}

type workflowClient struct {
	restClient rest.Interface
	ns         string
}

func (c *workflowClient) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	result := wfv1.WorkflowList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *workflowClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.Workflow, error) {
	result := wfv1.Workflow{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *workflowClient) Create(ctx context.Context, project *wfv1.Workflow) (*wfv1.Workflow, error) {
	result := wfv1.Workflow{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(resourceName).
		Body(project).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *workflowClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func UnmarshalWorkflow(wfStr string, strict bool) (wfv1.Workflow, error) {
	wfs, err := UnmarshalWorkflows(wfStr, strict)
	if err != nil {
		return wfv1.Workflow{}, err
	}
	for _, v := range wfs {
		return v, nil
	}
	return wfv1.Workflow{}, nil
}

func UnmarshalWorkflows(wfStr string, strict bool) ([]wfv1.Workflow, error) {
	wfBytes := []byte(wfStr)
	return wfcommon.SplitWorkflowYAMLFile(wfBytes, strict)
}

func GetDefaultWorkflow() (*wfv1.Workflow, error) {
	content, err := utils.ReadFile(getWorkflowTemplatePath())
	if err != nil {
		return nil, err
	}
	wf, err := UnmarshalWorkflow(content, true)
	return &wf, err
}

func GetDefaultWorkflowStr() (string, error) {
	return utils.ReadFile(getWorkflowTemplatePath())
}

func getWorkflowTemplatePath() string {
	return "pkg/argoworkflows/workflow-template.yaml"
}
