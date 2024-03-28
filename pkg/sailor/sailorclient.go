package sailor

import (
	"fmt"

	sailorv1alpha1 "github.com/f-rambo/sailor/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type AppV1Alpha1Interface interface {
	Apps(namespace string) AppInterface
}

type AppV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*AppV1Alpha1Client, error) {
	sailorv1alpha1.AddToScheme(scheme.Scheme)
	config := *c
	config.ContentConfig.GroupVersion = &sailorv1alpha1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &AppV1Alpha1Client{restClient: client}, nil
}

func (c *AppV1Alpha1Client) Apps(namespace string) AppInterface {
	return &appClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func GetApiVersion() string {
	return fmt.Sprintf("%s/%s", sailorv1alpha1.GroupVersion.Group, sailorv1alpha1.GroupVersion.Version)
}

func GetKind() string {
	return "App"
}

func BuildAppResource(namespace, name string, appSpec sailorv1alpha1.AppSpec) sailorv1alpha1.App {
	return sailorv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GetApiVersion(),
			Kind:       GetKind(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appSpec,
	}
}
