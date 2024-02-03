package data

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type clusterRepo struct {
	data *Data
	log  *log.Helper
}

func NewClusterRepo(data *Data, logger log.Logger) biz.ClusterRepo {
	return &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (c *clusterRepo) Save(ctx context.Context, cluster *biz.Cluster) error {
	// 开始事务
	tx := c.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 保存集群信息
	err := tx.Model(&biz.Cluster{}).Save(cluster).Error
	if err != nil {
		return err
	}
	// 保存节点信息
	for _, node := range cluster.Nodes {
		node.ClusterID = cluster.ID
		err = tx.Model(&biz.Node{}).Save(node).Error
		if err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

func (c *clusterRepo) Get(ctx context.Context, id int64) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("id = ?", id).First(cluster).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]*biz.Node, 0)
	err = c.data.db.Model(&biz.Node{}).Where("cluster_id = ?", cluster.ID).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	cluster.Nodes = append(cluster.Nodes, nodes...)
	return cluster, nil
}

func (c *clusterRepo) List(ctx context.Context) ([]*biz.Cluster, error) {
	var clusters []*biz.Cluster
	err := c.data.db.Model(&biz.Cluster{}).Find(&clusters).Error
	return clusters, err
}

func (c *clusterRepo) Delete(ctx context.Context, id int64) error {
	// 开始事务
	tx := c.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 删除集群信息
	err := tx.Model(&biz.Cluster{}).Where("id = ?", id).Delete(&biz.Cluster{}).Error
	if err != nil {
		return err
	}
	// 删除节点信息
	err = tx.Model(&biz.Node{}).Where("cluster_id = ?", id).Delete(&biz.Node{}).Error
	if err != nil {
		return err
	}
	return tx.Commit().Error
}
