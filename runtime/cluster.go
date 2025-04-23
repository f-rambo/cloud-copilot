package runtime

import (
	"context"
	"path/filepath"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CloudClusterKind = "CloudCluster"
)

type ClusterRuntime struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntime(conf *conf.Bootstrap, logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntime{
		conf: conf,
		log:  log.NewHelper(logger),
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

func (c *ClusterRuntime) Install(ctx context.Context, cluster *biz.Cluster) error {
	installYaml, err := utils.TransferredMeaning(
		cluster,
		filepath.Join(c.conf.Infrastructure.Component, Install),
	)
	if err != nil {
		return err
	}
	return CreateResourceByYaml(ctx, installYaml)
}
