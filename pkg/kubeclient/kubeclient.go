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

func GetKubeConfig() (config *rest.Config, err error) {
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
	config, err := GetKubeConfig()
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
