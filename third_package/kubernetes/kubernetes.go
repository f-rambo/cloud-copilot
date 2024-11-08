package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeClientByKubeConfig(KubeConfigPath string) (clientset *kubernetes.Clientset, err error) {
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

func GetKubeClientByInCluster() (clientset *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func GetKubeClientByRestConfig(masterIp, token, ca, key, cert string) (clientset *kubernetes.Clientset, err error) {
	config := &rest.Config{
		Host:        masterIp + ":6443",
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(ca),
			KeyData:  []byte(key),
			CertData: []byte(cert),
		},
	}
	return kubernetes.NewForConfig(config)
}
