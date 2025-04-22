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
	if cluster.Id == 0 {
		err = tx.Model(&biz.Cluster{}).Create(cluster).Error
	} else {
		err = tx.Model(&biz.Cluster{}).Where("id =?", cluster.Id).Updates(cluster).Error
	}
	if err != nil {
		return err
	}
	funcs := []func(context.Context, *biz.Cluster, *gorm.DB) error{
		c.saveNodeGroup,
		c.saveNode,
		c.saveCloudResources,
		c.saveSecuritys,
	}
	for _, f := range funcs {
		getErr := f(ctx, cluster, tx)
		if getErr != nil {
			return getErr
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

func (c *clusterRepo) saveSecuritys(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, v := range cluster.Securitys {
		v.ClusterId = cluster.Id
		err := tx.Model(&biz.Security{}).Where("id = ?", v.Id).Save(v).Error
		if err != nil {
			return err
		}
	}
	sgs := make([]*biz.Security, 0)
	err := tx.Model(&biz.Security{}).Where("cluster_id = ?", cluster.Id).Find(&sgs).Error
	if err != nil {
		return err
	}
	for _, v := range sgs {
		isExist := false
		for _, v1 := range cluster.Securitys {
			if v.Id == v1.Id {
				isExist = true
				break
			}
		}
		if !isExist {
			err := tx.Model(&biz.Security{}).Where("id = ?", v.Id).Delete(v).Error
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
	securitys := make([]*biz.Security, 0)
	err = c.data.db.Model(&biz.Security{}).Where("cluster_id = ?", cluster.Id).Find(&securitys).Error
	if err != nil {
		return nil, err
	}
	if len(securitys) != 0 {
		cluster.Securitys = securitys
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

func (c *clusterRepo) List(ctx context.Context, name string, page, pageSize int32) ([]*biz.Cluster, int64, error) {
	var clusters []*biz.Cluster
	var total int64

	query := c.data.db.Model(&biz.Cluster{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(int(offset)).Limit(int(pageSize)).Find(&clusters).Error
	if err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
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
	err = tx.Model(&biz.Security{}).Where("cluster_id = ?", id).Delete(&biz.Security{}).Error
	if err != nil {
		return err
	}
	return tx.Commit().Error
}
