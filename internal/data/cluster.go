package data

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type clusterRepo struct {
	data *Data
	log  *log.Helper
}

func NewClusterRepo(data *Data, logger log.Logger) (biz.ClusterRepo, error) {
	repo := &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
	return repo, repo.init()
}

func (c *clusterRepo) init() error {
	if c.data.kubeClient == nil {
		return nil
	}
	var count int64
	err := c.data.db.Model(&biz.Cluster{}).Where("name = ?", "default").Count(&count).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if count > 0 {
		return nil
	}
	// 获取集群信息
	versionInfo, err := c.data.kubeClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	// api server 地址
	serverAddress := c.data.kubeClient.Discovery().RESTClient().Get().URL().Host
	cluster := &biz.Cluster{
		Name:             "default",
		ServerVersion:    versionInfo.String(),
		ApiServerAddress: serverAddress,
	}
	err = c.data.db.Model(&biz.Cluster{}).Create(cluster).Error
	if err != nil {
		return err
	}
	// 获取节点信息
	nodeList, err := c.data.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		labelsJson, err := json.Marshal(node.Labels)
		if err != nil {
			return err
		}
		annotationsJson, err := json.Marshal(node.Annotations)
		if err != nil {
			return err
		}
		roles := make([]string, 0)
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			roles = append(roles, "master")
		}
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			roles = append(roles, "master")
		}
		if _, ok := node.Labels["node-role.kubernetes.io/edge"]; ok {
			roles = append(roles, "edge")
		}
		if _, ok := node.Labels["node-role.kubernetes.io/worker"]; ok {
			roles = append(roles, "worker")
		}
		if len(roles) == 0 {
			roles = append(roles, "worker")
		}
		roleJson, err := json.Marshal(roles)
		if err != nil {
			return err
		}
		var internalIP string
		var externalIP string
		for _, addr := range node.Status.Addresses {
			if addr.Type == coreV1.NodeInternalIP {
				internalIP = addr.Address
			}
			if addr.Type == coreV1.NodeExternalIP {
				externalIP = addr.Address
			}
		}
		cluster.Nodes = append(cluster.Nodes, &biz.Node{
			Name:        node.Name,
			Labels:      string(labelsJson),
			Annotations: string(annotationsJson),
			OSImage:     node.Status.NodeInfo.OSImage,
			Kernel:      node.Status.NodeInfo.KernelVersion,
			Container:   node.Status.NodeInfo.ContainerRuntimeVersion,
			Kubelet:     node.Status.NodeInfo.KubeletVersion,
			KubeProxy:   node.Status.NodeInfo.KubeProxyVersion,
			InternalIP:  internalIP,
			ExternalIP:  externalIP,
			Role:        string(roleJson),
			ClusterID:   cluster.ID,
		})
	}
	err = c.data.db.Model(&biz.Node{}).Create(&cluster.Nodes).Error
	if err != nil {
		return err
	}
	return nil
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

func (c *clusterRepo) ClusterClient(ctx context.Context, clusterID int64) (*kubernetes.Clientset, error) {
	// todo 来区分不同的集群
	return c.data.kubeClient, nil
}
