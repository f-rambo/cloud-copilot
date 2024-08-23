package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeClient(KubeConfigPath string) (clientset *kubernetes.Clientset, err error) {
	if KubeConfigPath == "" {
		KubeConfigPath = clientcmd.RecommendedHomeFile
	}
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}
