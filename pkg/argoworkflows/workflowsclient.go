package argoworkflows

import (
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type WorkflowV1Alpha1Interface interface {
	Workflows(namespace string) WorkflowInterface
}

type WorkflowV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*WorkflowV1Alpha1Client, error) {
	wfv1.AddToScheme(scheme.Scheme)
	config := *c
	config.ContentConfig.GroupVersion = &wfv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &WorkflowV1Alpha1Client{restClient: client}, nil
}

func (c *WorkflowV1Alpha1Client) Workflows(namespace string) WorkflowInterface {
	return &workflowClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func GetApiVersion() string {
	return fmt.Sprintf("%s/%s", wfv1.WorkflowSchemaGroupVersionKind.Group, wfv1.WorkflowSchemaGroupVersionKind.Version)
}

func GetKind() string {
	return wfv1.WorkflowSchemaGroupVersionKind.Kind
}
