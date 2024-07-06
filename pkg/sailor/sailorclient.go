package sailor

import (
	"fmt"

	sailorv1alpha1 "github.com/f-rambo/sailor/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type AppV1Alpha1Interface interface {
	Apps(namespace string) AppInterface
}

type AppV1Alpha1Client struct {
	restClient rest.Interface
}

func getKubeConfig() (config *rest.Config, err error) {
	config, err = rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return
		}
	}
	return
}

func getKubeClientSet() (clientset *kubernetes.Clientset, err error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func newForConfig(c *rest.Config) (*AppV1Alpha1Client, error) {
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

func (c *AppV1Alpha1Client) apps(namespace string) AppInterface {
	return &appClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func getApiVersion() string {
	return fmt.Sprintf("%s/%s", sailorv1alpha1.GroupVersion.Group, sailorv1alpha1.GroupVersion.Version)
}

func getKind() string {
	return "App"
}

func buildAppResource(namespace, name string, appSpec sailorv1alpha1.AppSpec) sailorv1alpha1.App {
	return sailorv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: getApiVersion(),
			Kind:       getKind(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appSpec,
	}
}
