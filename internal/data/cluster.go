package data

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type clusterRepo struct {
	data *Data
	log  *log.Helper
	c    *conf.Bootstrap
}

func NewClusterRepo(data *Data, c *conf.Bootstrap, logger log.Logger) biz.ClusterData {
	return &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
		c:    c,
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
		c.saveBostionHost,
		c.saveNodeGroup,
		c.saveNode,
		c.saveCloudResources,
		c.saveSecurityGroup,
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

func (c *clusterRepo) saveBostionHost(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	if cluster.BostionHost == nil {
		bostionHosts := make([]*biz.BostionHost, 0)
		err := tx.Model(&biz.BostionHost{}).Where("cluster_id = ?", cluster.Id).Find(&bostionHosts).Error
		if err != nil {
			return err
		}
		for _, bostionHost := range bostionHosts {
			if bostionHost.Id == "" {
				continue
			}
			err = tx.Model(&biz.BostionHost{}).Where("id = ?", bostionHost.Id).Delete(bostionHost).Error
			if err != nil {
				return err
			}
		}
		return nil
	}
	cluster.BostionHost.ClusterId = cluster.Id
	err := tx.Model(&biz.BostionHost{}).Where("id = ?", cluster.BostionHost.Id).Save(cluster.BostionHost).Error
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
		for _, subCloudResource := range cloudResource.SubResources {
			subCloudResource.ClusterId = cluster.Id
			subCloudResource.ParentId = cloudResource.Id
			err := tx.Model(&biz.CloudResource{}).Where("id = ?", subCloudResource.Id).Save(subCloudResource).Error
			if err != nil {
				return err
			}
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
			for _, subCloudResource := range cloudResource.SubResources {
				if subCloudResource.Id == v.Id {
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			err := tx.Delete(&biz.CloudResource{}, v.Id).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) saveSecurityGroup(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, v := range cluster.SecurityGroups {
		v.ClusterId = cluster.Id
		err := tx.Model(&biz.SecurityGroup{}).Where("id = ?", v.Id).Save(v).Error
		if err != nil {
			return err
		}
	}
	sgs := make([]*biz.SecurityGroup, 0)
	err := tx.Model(&biz.SecurityGroup{}).Where("cluster_id = ?", cluster.Id).Find(&sgs).Error
	if err != nil {
		return err
	}
	for _, v := range sgs {
		isExist := false
		for _, v1 := range cluster.SecurityGroups {
			if v.Id == v1.Id {
				isExist = true
			}
		}
		if !isExist {
			err := tx.Model(&biz.SecurityGroup{}).Where("id = ?", v.Id).Delete(v).Error
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
	bostionHost := &biz.BostionHost{}
	err = c.data.db.Model(&biz.BostionHost{}).Where("cluster_id = ?", cluster.Id).First(bostionHost).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if bostionHost.Id != "" {
		cluster.BostionHost = bostionHost
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
		data := make([]*biz.CloudResource, 0)
		for _, v := range cloudResources {
			if v.ParentId != "" {
				continue
			}
			data = append(data, v)
		}
		for _, v := range data {
			subCloudResources := make([]*biz.CloudResource, 0)
			for _, v1 := range cloudResources {
				if v.Id != "" && v1.ParentId == v.Id {
					subCloudResources = append(subCloudResources, v1)
				}
			}
			v.SubResources = subCloudResources
		}
		cluster.CloudResources = data
	}
	sgs := make([]*biz.SecurityGroup, 0)
	err = c.data.db.Model(&biz.SecurityGroup{}).Where("cluster_id = ?", cluster.Id).Find(&sgs).Error
	if err != nil {
		return nil, err
	}
	if len(sgs) != 0 {
		cluster.SecurityGroups = sgs
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
	if cluster.Version != "" {
		clusterModelObj = clusterModelObj.Where("server_version = ?", cluster.Version)
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
	err = tx.Model(&biz.BostionHost{}).Where("cluster_id = ?", id).Delete(&biz.BostionHost{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.CloudResource{}).Where("cluster_id = ?", id).Delete(&biz.CloudResource{}).Error
	if err != nil {
		return err
	}
	err = tx.Model(&biz.SecurityGroup{}).Where("cluster_id = ?", id).Delete(&biz.SecurityGroup{}).Error
	if err != nil {
		return err
	}
	return tx.Commit().Error
}
