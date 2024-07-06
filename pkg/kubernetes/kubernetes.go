package kubernetes

import (
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

func getKubeConfig(args *ConfigArgs) (config *rest.Config, err error) {
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

func getKubeClientSet() (clientset *kubernetes.Clientset, err error) {
	config, err := getKubeConfig(nil)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
