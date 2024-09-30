package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/f-rambo/ocean/utils"
)

// inherit from biz cluster

func (c *Cluster) generateNodeLables(nodeGroup *NodeGroup) string {
	lableMap := make(map[string]string)
	lableMap["cluster"] = c.Name
	lableMap["cluster_type"] = c.Type.String()
	lableMap["region"] = c.Region
	lableMap["nodegroup"] = nodeGroup.Name
	lableMap["nodegroup_type"] = nodeGroup.Type.String()
	lablebytes, _ := json.Marshal(lableMap)
	return string(lablebytes)
}

// 获取当前集群最新信息
func (uc *ClusterUsecase) GetCurrentCluster(ctx context.Context) (*Cluster, error) {
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// 根据nodegroup增加节点
func (uc *ClusterUsecase) NodeGroupIncreaseSize(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup, size int32) error {
	for i := 0; i < int(size); i++ {
		node := &Node{
			Name:        fmt.Sprintf("%s-%s", cluster.Name, utils.GetRandomString()),
			Role:        NodeRoleWorker,
			Status:      NodeStatusCreating,
			ClusterID:   cluster.ID,
			NodeGroupID: nodeGroup.ID,
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}
	return uc.Apply(ctx, cluster)
}

// 删除节点
func (uc *ClusterUsecase) DeleteNodes(ctx context.Context, cluster *Cluster, nodes []*Node) error {
	for _, node := range nodes {
		for i, n := range cluster.Nodes {
			if n.ID == node.ID {
				cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
				break
			}
		}
	}
	return uc.Apply(ctx, cluster)
}

// 预测一个节点配置，也就是根据当前节点组目前还可以配置的节点
func (uc *ClusterUsecase) NodeGroupTemplateNodeInfo(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup) (*Node, error) {
	return &Node{
		Name:        fmt.Sprintf("%s-%s", cluster.Name, utils.GetRandomString()),
		Role:        NodeRoleWorker,
		Status:      NodeStatusCreating,
		ClusterID:   cluster.ID,
		NodeGroupID: nodeGroup.ID,
		Labels:      cluster.generateNodeLables(nodeGroup),
	}, nil
}

// 在云提供商销毁前清理打开的资源，例如协程等
func (uc *ClusterUsecase) Cleanup(ctx context.Context) error {
	return nil
}

// 在每个主循环前调用，用于动态更新云提供商状态
func (uc *ClusterUsecase) Refresh(ctx context.Context) error {
	// 获取当前集群状态更新状态
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return err
	}
	cluster, err = uc.clusterRepo.GetByName(ctx, cluster.Name)
	if err != nil {
		return err
	}
	for _, v := range cluster.Nodes {
		for _, currentNode := range cluster.Nodes {
			if v.Name == currentNode.Name {
				v.Status = currentNode.Status
				break
			}
		}
	}
	err = uc.Save(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}
