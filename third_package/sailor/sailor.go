package sailor

import (
	"context"

	sailorV1alpha1 "github.com/f-rambo/sailor/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	resourceName = "apps"
)

type AppInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*sailorV1alpha1.AppList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*sailorV1alpha1.App, error)
	Create(context.Context, *sailorV1alpha1.App) (*sailorV1alpha1.App, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string) error
	Update(ctx context.Context, app *sailorV1alpha1.App) (*sailorV1alpha1.App, error)
	// ...
}

type appClient struct {
	restClient rest.Interface
	ns         string
}

func (c *appClient) List(ctx context.Context, opts metav1.ListOptions) (*sailorV1alpha1.AppList, error) {
	result := sailorV1alpha1.AppList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *appClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*sailorV1alpha1.App, error) {
	result := sailorV1alpha1.App{}
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

func (c *appClient) Create(ctx context.Context, app *sailorV1alpha1.App) (*sailorV1alpha1.App, error) {
	result := sailorV1alpha1.App{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(resourceName).
		Body(app).
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

func (c *appClient) Update(ctx context.Context, app *sailorV1alpha1.App) (*sailorV1alpha1.App, error) {
	// app.ResourceVersion = utils.GetUUID()
	result := &sailorV1alpha1.App{}
	err := c.restClient.
		Put().Namespace(c.ns).
		Resource(resourceName).
		Name(app.Name).
		Body(app).
		Do(ctx).
		Into(result)

	return result, err
}
