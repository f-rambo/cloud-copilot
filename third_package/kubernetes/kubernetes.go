package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
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
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes by kubeconfig client failed")
	}
	return client, nil
}

func GetKubeClientByInCluster() (clientset *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes by in cluster client failed")
	}
	return client, nil
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
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes by rest config client failed")
	}
	return client, nil
}

func GetDynamicClientByRestConfig(masterIp, token, ca, key, cert string) (*dynamic.DynamicClient, error) {
	config := &rest.Config{
		Host:        masterIp + ":6443",
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(ca),
			KeyData:  []byte(key),
			CertData: []byte(cert),
		},
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get dynamic client by rest config failed")
	}
	return client, nil
}
