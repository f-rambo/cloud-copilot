package kubeclient

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ConfigArgs struct {
	ApiServer string
	Token     string
	CaData    string
	KeyData   string
	CertData  string
}

func GetKubeConfig(args *ConfigArgs) (config *rest.Config, err error) {
	if args != nil {
		config := &rest.Config{
			Host:        args.ApiServer,
			BearerToken: args.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(args.CaData),
				KeyData:  []byte(args.KeyData),
				CertData: []byte(args.CertData),
			},
		}
		return config, nil
	}
	config, err = rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return
		}
	}
	return
}

func GetKubeClientSet() (clientset *kubernetes.Clientset, err error) {
	config, err := GetKubeConfig(nil)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func NamespaceExists(ctx context.Context, kubeClientSet *kubernetes.Clientset, namespaceName string) (bool, error) {
	_, err := kubeClientSet.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func CreateNamespace(ctx context.Context, kubeClientSet *kubernetes.Clientset, namespaceName string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	_, err := kubeClientSet.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	return err
}
