package operatorapp

import (
	"context"

	operatoroceaniov1alpha1 "github.com/f-rambo/operatorapp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	resourceName = "apps"
)

type AppInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*operatoroceaniov1alpha1.AppList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*operatoroceaniov1alpha1.App, error)
	Create(context.Context, *operatoroceaniov1alpha1.App) (*operatoroceaniov1alpha1.App, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string) error
	// ...
}

type appClient struct {
	restClient rest.Interface
	ns         string
}

func (c *appClient) List(ctx context.Context, opts metav1.ListOptions) (*operatoroceaniov1alpha1.AppList, error) {
	result := operatoroceaniov1alpha1.AppList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *appClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*operatoroceaniov1alpha1.App, error) {
	result := operatoroceaniov1alpha1.App{}
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

func (c *appClient) Create(ctx context.Context, project *operatoroceaniov1alpha1.App) (*operatoroceaniov1alpha1.App, error) {
	result := operatoroceaniov1alpha1.App{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(resourceName).
		Body(project).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *appClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

func (c *appClient) Delete(ctx context.Context, name string) error {
	return c.restClient.
		Delete().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		Do(ctx).
		Error()
}
