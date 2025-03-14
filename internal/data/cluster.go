package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type clusterRepo struct {
	data *Data
	log  *log.Helper
}

func NewClusterRepo(data *Data, logger log.Logger) biz.ClusterData {
	return &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (c *clusterRepo) Save(ctx context.Context, cluster *biz.Cluster) (err error) {
	tx := c.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()
	err = tx.Model(&biz.Cluster{}).Where("id = ?", cluster.Id).Save(cluster).Error
	if err != nil {
		return err
	}
	funcs := []func(context.Context, *biz.Cluster, *gorm.DB) error{
		c.saveNodeGroup,
		c.saveNode,
		c.saveCloudResources,
		c.saveIngressControllerRules,
	}
	for _, f := range funcs {
		err := f(ctx, cluster, tx)
		if err != nil {
			return err
		}
	}
	err = tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func (c *clusterRepo) saveNodeGroup(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, nodeGroup := range cluster.NodeGroups {
		nodeGroup.ClusterId = cluster.Id
		err := tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.Id).Save(nodeGroup).Error
		if err != nil {
			return err
		}
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	err := tx.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.Id).Find(&nodeGroups).Error
	if err != nil {
		return err
	}
	for _, nodeGroup := range nodeGroups {
		ok := false
		for _, nodeGroup2 := range cluster.NodeGroups {
			if nodeGroup.Id == nodeGroup2.Id {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.Id).Delete(nodeGroup).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) saveNode(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, node := range cluster.Nodes {
		node.ClusterId = cluster.Id
		err := tx.Model(&biz.Node{}).Where("id = ?", node.Id).Save(node).Error
		if err != nil {
			return err
		}
	}
	nodes := make([]*biz.Node, 0)
	err := tx.Model(&biz.Node{}).Where("cluster_id = ?", cluster.Id).Find(&nodes).Error
	if err != nil {
		return err
	}
	for _, node := range nodes {
		ok := false
		for _, node2 := range cluster.Nodes {
			if node.Id == node2.Id {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.Node{}).Where("id = ?", node.Id).Delete(node).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) saveCloudResources(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, cloudResource := range cluster.CloudResources {
		cloudResource.ClusterId = cluster.Id
		err := tx.Model(&biz.CloudResource{}).Where("id = ?", cloudResource.Id).Save(cloudResource).Error
		if err != nil {
			return err
		}
	}
	cloudResources := make([]*biz.CloudResource, 0)
	err := tx.Model(&biz.CloudResource{}).Where("cluster_id = ?", cluster.Id).Find(&cloudResources).Error
	if err != nil {
		return err
	}
	for _, v := range cloudResources {
		ok := false
		for _, cloudResource := range cluster.CloudResources {
			if v.Id == cloudResource.Id {
				ok = true
				break
			}
		}
		if !ok {
			err := tx.Model(&biz.CloudResource{}).Where("id = ?", v.Id).Delete(v).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) saveIngressControllerRules(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, v := range cluster.IngressControllerRules {
		v.ClusterId = cluster.Id
		err := tx.Model(&biz.IngressControllerRule{}).Where("id = ?", v.Id).Save(v).Error
		if err != nil {
			return err
		}
	}
	sgs := make([]*biz.IngressControllerRule, 0)
	err := tx.Model(&biz.IngressControllerRule{}).Where("cluster_id = ?", cluster.Id).Find(&sgs).Error
	if err != nil {
		return err
	}
	for _, v := range sgs {
		isExist := false
		for _, v1 := range cluster.IngressControllerRules {
			if v.Id == v1.Id {
				isExist = true
				break
			}
		}
		if !isExist {
			err := tx.Model(&biz.IngressControllerRule{}).Where("id = ?", v.Id).Delete(v).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) Get(ctx context.Context, id int64) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("id = ?", id).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if cluster.Id == 0 {
		return nil, nil
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	err = c.data.db.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.Id).Find(&nodeGroups).Error
	if err != nil {
		return nil, err
	}
	if len(nodeGroups) != 0 {
		cluster.NodeGroups = nodeGroups
	}
	nodes := make([]*biz.Node, 0)
	err = c.data.db.Model(&biz.Node{}).Where("cluster_id = ?", cluster.Id).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	if len(nodes) != 0 {
		cluster.Nodes = nodes
	}
	cloudResources := make([]*biz.CloudResource, 0)
	err = c.data.db.Model(&biz.CloudResource{}).Where("cluster_id = ?", cluster.Id).Find(&cloudResources).Error
	if err != nil {
		return nil, err
	}
	if len(cloudResources) != 0 {
		cluster.CloudResources = cloudResources
	}
	ingressRules := make([]*biz.IngressControllerRule, 0)
	err = c.data.db.Model(&biz.IngressControllerRule{}).Where("cluster_id = ?", cluster.Id).Find(&ingressRules).Error
	if err != nil {
		return nil, err
	}
	if len(ingressRules) != 0 {
		cluster.IngressControllerRules = ingressRules
	}
	return cluster, nil
}

func (c *clusterRepo) GetByName(ctx context.Context, name string) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("name = ?", name).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if cluster.Id != 0 {
		return c.Get(ctx, cluster.Id)
	}
	return cluster, nil
}

func (c *clusterRepo) List(ctx context.Context, cluster *biz.Cluster) ([]*biz.Cluster, error) {
	var clusters []*biz.Cluster
	clusterModelObj := c.data.db.Model(&biz.Cluster{})
	if cluster == nil {
		err := clusterModelObj.Find(&clusters).Error
		return clusters, err
	}
	if cluster.Id != 0 {
		clusterModelObj = clusterModelObj.Where("id = ?", cluster.Id)
	}
	if cluster.Name != "" {
		clusterModelObj = clusterModelObj.Where("name = ?", cluster.Name)
	}
	if cluster.KuberentesVersion != "" {
		clusterModelObj = clusterModelObj.Where("server_version = ?", cluster.KuberentesVersion)
	}
	err := clusterModelObj.Find(&clusters).Error
	return clusters, err
}

func (c *clusterRepo) Delete(ctx context.Context, id int64) (err error) {
	tx := c.data.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	err = tx.Model(&biz.Cluster{}).Where("id = ?", id).Delete(&biz.Cluster{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.Node{}).Where("cluster_id = ?", id).Delete(&biz.Node{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.NodeGroup{}).Where("cluster_id = ?", id).Delete(&biz.NodeGroup{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.CloudResource{}).Where("cluster_id = ?", id).Delete(&biz.CloudResource{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.IngressControllerRule{}).Where("cluster_id = ?", id).Delete(&biz.IngressControllerRule{}).Error
	if err != nil {
		return err
	}
	return tx.Commit().Error
}

func (c *clusterRepo) GetClusterAppReleaseByName(ctx context.Context, name string) (*biz.AppRelease, error) {
	appRelease := &biz.AppRelease{}
	err := c.data.db.Model(&biz.AppRelease{}).Where("release_name = ?", name).First(appRelease).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return appRelease, nil
}
