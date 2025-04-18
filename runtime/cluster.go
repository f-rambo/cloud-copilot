package runtime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CloudClusterKind = "CloudCluster"
)

type ClusterRuntime struct {
	log *log.Helper
}

func NewClusterRuntime(logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntime{
		log: log.NewHelper(logger),
	}
}

func (c *ClusterRuntime) ReloadCluster(ctx context.Context, cluster *biz.Cluster) error {
	obj := NewUnstructured(CloudClusterKind)
	obj.SetName(cluster.Name)
	SetSpec(obj, cluster)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return biz.ErrClusterNotFound
	}
	_, err = GetResource(ctx, dynamicClient, obj)
	if k8sErr.IsNotFound(err) {
		err = CreateResource(ctx, dynamicClient, obj)
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	err = UpdateResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntime) CurrentCluster(ctx context.Context, cluster *biz.Cluster) error {
	obj := NewUnstructured(CloudClusterKind)
	obj.SetName(cluster.Name)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return biz.ErrClusterNotFound
	}
	res, err := GetResource(ctx, dynamicClient, obj)
	if err != nil && k8sErr.IsNotFound(err) {
		return biz.ErrClusterNotFound
	}
	err = GetSpec(res, cluster)
	if err != nil {
		return err
	}
	return nil
}
