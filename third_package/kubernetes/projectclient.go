package kubernetes

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectClient struct {
	c   *conf.Bootstrap
	log *log.Helper
}

func NewProjectClient(c *conf.Bootstrap, logger log.Logger) biz.ClusterPorjectRepo {
	return &ProjectClient{
		c:   c,
		log: log.NewHelper(logger),
	}
}

func (p *ProjectClient) CreateNamespace(ctx context.Context, namespace string) error {
	kubeClientSet, err := GetKubeClientByInCluster()
	if err != nil {
		return err
	}
	_, err = kubeClientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	return err
}

func (p *ProjectClient) GetNamespaces(ctx context.Context) (namespaces []string, err error) {
	namespaces = make([]string, 0)
	kubeClientSet, err := GetKubeClientByInCluster()
	if err != nil {
		return
	}
	namespaceList, err := kubeClientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, namespace := range namespaceList.Items {
		namespaces = append(namespaces, namespace.Name)
	}
	return namespaces, nil
}
