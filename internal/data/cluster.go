package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
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
	cluster.Type = strings.ToLower(cluster.Type)
	// 开始事务
	tx := c.data.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 保存集群信息
	err := tx.Model(&biz.Cluster{}).Where("id = ?", cluster.ID).Save(cluster).Error
	if err != nil {
		return err
	}
	// 保存节点信息
	nodes := make([]*biz.Node, 0)
	err = tx.Model(&biz.Node{}).Where("cluster_id = ?", cluster.ID).Find(&nodes).Error
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
	for _, node := range cluster.Nodes {
		node.ClusterID = cluster.ID
		err = tx.Model(&biz.Node{}).Where("id = ?", node.ID).Save(node).Error
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

func (c *clusterRepo) GetByName(ctx context.Context, name string) (*biz.Cluster, error) {
	cluster := &biz.Cluster{}
	err := c.data.db.Model(&biz.Cluster{}).Where("name = ?", name).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return cluster, nil
}

func (c *clusterRepo) ReadClusterLog(cluster *biz.Cluster) error {
	clog := c.c.GetOceanLog()
	logPath := fmt.Sprintf("%s/cluster-%d.log", clog.GetPath(), cluster.ID)
	if utils.IsFileExist(logPath) {
		logs, err := utils.ReadFile(logPath)
		if err != nil {
			return err
		}
		cluster.Logs = string(logs)
	}
	return nil
}

func (c *clusterRepo) WriteClusterLog(cluster *biz.Cluster) error {
	clog := c.c.GetOceanLog()
	file, err := utils.NewFile(clog.GetPath(),
		fmt.Sprintf("cluster-%d.log", cluster.ID), true)
	if err != nil {
		return err
	}
	defer file.Close()
	err = file.Write([]byte(cluster.Logs))
	if err != nil {
		return err
	}
	cluster.Logs = ""
	return nil
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
	if cluster.ServerVersion != "" {
		clusterModelObj = clusterModelObj.Where("server_version = ?", cluster.ServerVersion)
	}
	err := clusterModelObj.Find(&clusters).Error
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

func (c *clusterRepo) Put(ctx context.Context, cluster *biz.Cluster) error {
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	return c.data.Put(ctx, ClusterQueueKey.String(), string(clusterJson))
}

func (c *clusterRepo) GetByQueue(ctx context.Context) (*biz.Cluster, error) {
	data, err := c.data.Get(ctx, ClusterQueueKey.String())
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

func (c *clusterRepo) DeleteByQueue(ctx context.Context) error {
	return c.data.Delete(ctx, ClusterQueueKey.String())
}
