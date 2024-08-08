package argoworkflows

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

/*
	argo workflow 资源
	customresourcedefinition.apiextensions.k8s.io/clusterworkflowtemplates.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/cronworkflows.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workflowartifactgctasks.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workfloweventbindings.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workflows.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workflowtaskresults.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workflowtasksets.argoproj.io created
	customresourcedefinition.apiextensions.k8s.io/workflowtemplates.argoproj.io created
	serviceaccount/argo created
	serviceaccount/argo-server created
	role.rbac.authorization.k8s.io/argo-role created
	clusterrole.rbac.authorization.k8s.io/argo-aggregate-to-admin created
	clusterrole.rbac.authorization.k8s.io/argo-aggregate-to-edit created
	clusterrole.rbac.authorization.k8s.io/argo-aggregate-to-view created
	clusterrole.rbac.authorization.k8s.io/argo-cluster-role created
	clusterrole.rbac.authorization.k8s.io/argo-server-cluster-role created
	rolebinding.rbac.authorization.k8s.io/argo-binding created
	clusterrolebinding.rbac.authorization.k8s.io/argo-binding created
	clusterrolebinding.rbac.authorization.k8s.io/argo-server-binding created
	configmap/workflow-controller-configmap created
	service/argo-server created
	priorityclass.scheduling.k8s.io/workflow-controller created
	deployment.apps/argo-server created
	deployment.apps/workflow-controller created
*/

/*
	argo cli
	kubectl create -n argo -f https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/hello-world.yaml
	kubectl get wf -n argo
	kubectl get wf hello-world-xxx -n argo
	kubectl get po -n argo --selector=workflows.argoproj.io/workflow=hello-world-xxx
	kubectl logs hello-world-yyy -c main -n argo
*/

var (
	resourceName = "workflows"
)

type WorkflowInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*wfv1.Workflow, error)
	Create(context.Context, *wfv1.Workflow) (*wfv1.Workflow, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string) error
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

func (c *workflowClient) Create(ctx context.Context, wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	result := wfv1.Workflow{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(resourceName).
		Body(wf).
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

func (c *workflowClient) Delete(ctx context.Context, name string) error {
	return c.restClient.
		Delete().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		Do(ctx).
		Error()
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
