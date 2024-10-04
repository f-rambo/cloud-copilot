package data

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

const ClusterQueueKey QueueKey = "ocean-cluster-queue"

type clusterRepo struct {
	data *Data
	log  *log.Helper
	c    *conf.Bootstrap
}

func NewClusterRepo(data *Data, c *conf.Bootstrap, logger log.Logger) biz.ClusterRepo {
	return &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
		c:    c,
	}
}

func (c *clusterRepo) Save(ctx context.Context, cluster *biz.Cluster) error {
	tx := c.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if len(cluster.CloudResources) != 0 {
		cloudResourceJsonByte, err := json.Marshal(cluster.CloudResources)
		if err != nil {
			return err
		}
		cluster.CloudResourcesJson = string(cloudResourceJsonByte)
	}
	err := tx.Model(&biz.Cluster{}).Where("id = ?", cluster.ID).Save(cluster).Error
	if err != nil {
		return err
	}

	err = c.saveBostionHost(ctx, cluster, tx)
	if err != nil {
		return err
	}

	err = c.saveNodeGroup(ctx, cluster, tx)
	if err != nil {
		return err
	}

	err = c.saveNode(ctx, cluster, tx)
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func (c *clusterRepo) saveBostionHost(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	bostionHost := &biz.BostionHost{}
	err := tx.Model(&biz.BostionHost{}).Where("cluster_id = ?", cluster.ID).First(bostionHost).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if bostionHost.ID != 0 && cluster.BostionHost != nil {
		cluster.BostionHost.ID = bostionHost.ID
	}
	if bostionHost.ID != 0 && cluster.BostionHost == nil {
		err = tx.Model(&biz.BostionHost{}).Where("cluster_id = ?", cluster.ID).Delete(&biz.BostionHost{}).Error
		if err != nil {
			return err
		}
	}
	if cluster.BostionHost != nil {
		cluster.BostionHost.ClusterID = cluster.ID
		err = tx.Model(&biz.BostionHost{}).Where("id = ?", cluster.BostionHost.ID).Save(cluster.BostionHost).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterRepo) saveNodeGroup(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, nodeGroup := range cluster.NodeGroups {
		nodeGroup.ClusterID = cluster.ID
		err := tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.ID).Save(nodeGroup).Error
		if err != nil {
			return err
		}
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	err := tx.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.ID).Find(&nodeGroups).Error
	if err != nil {
		return err
	}
	for _, nodeGroup := range nodeGroups {
		ok := false
		for _, nodeGroup2 := range cluster.NodeGroups {
			if nodeGroup.ID == nodeGroup2.ID {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.NodeGroup{}).Where("id = ?", nodeGroup.ID).Delete(nodeGroup).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *clusterRepo) saveNode(_ context.Context, cluster *biz.Cluster, tx *gorm.DB) error {
	for _, node := range cluster.Nodes {
		node.ClusterID = cluster.ID
		err := tx.Model(&biz.Node{}).Where("id = ?", node.ID).Save(node).Error
		if err != nil {
			return err
		}
	}
	nodes := make([]*biz.Node, 0)
	err := tx.Model(&biz.Node{}).Where("cluster_id = ?", cluster.ID).Find(&nodes).Error
	if err != nil {
		return err
	}
	for _, node := range nodes {
		ok := false
		for _, node2 := range cluster.Nodes {
			if node.ID == node2.ID {
				ok = true
				break
			}
		}
		if !ok {
			err = tx.Model(&biz.Node{}).Where("id = ?", node.ID).Delete(node).Error
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
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if cluster.CloudResourcesJson != "" {
		cluster.CloudResources = make(map[biz.ResourceType][]*biz.CloudResource)
		err = json.Unmarshal([]byte(cluster.CloudResourcesJson), &cluster.CloudResources)
		if err != nil {
			return nil, err
		}
	}
	bostionHost := &biz.BostionHost{}
	err = c.data.db.Model(&biz.BostionHost{}).Where("cluster_id = ?", cluster.ID).First(bostionHost).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	cluster.BostionHost = bostionHost
	nodeGroups := make([]*biz.NodeGroup, 0)
	err = c.data.db.Model(&biz.NodeGroup{}).Where("cluster_id = ?", cluster.ID).Find(&nodeGroups).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]*biz.Node, 0)
	err = c.data.db.Model(&biz.Node{}).Where("cluster_id = ?", cluster.ID).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	cluster.NodeGroups = append(cluster.NodeGroups, nodeGroups...)
	cluster.Nodes = append(cluster.Nodes, nodes...)
	return cluster, nil
}

func (c *clusterRepo) GetByName(ctx context.Context, name string) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("name = ?", name).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
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
	if cluster.ID != 0 {
		clusterModelObj = clusterModelObj.Where("id = ?", cluster.ID)
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

func (c *clusterRepo) Delete(ctx context.Context, id int64) error {
	tx := c.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	err := tx.Model(&biz.Cluster{}).Where("id = ?", id).Delete(&biz.Cluster{}).Error
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
	return tx.Commit().Error
}

func (c *clusterRepo) Put(ctx context.Context, cluster *biz.Cluster) error {
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	return c.data.Put(ctx, ClusterQueueKey.String(), string(clusterJson))
}

func (c *clusterRepo) Watch(ctx context.Context) (*biz.Cluster, error) {
	data, err := c.data.Watch(ctx, ClusterQueueKey.String())
	if err != nil {
		return nil, err
	}
	if data == "" {
		return nil, nil
	}
	cluster := &biz.Cluster{}
	err = json.Unmarshal([]byte(data), cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
