package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func (c *Cluster) generateNodeLables(nodeGroup *NodeGroup) string {
	lableMap := make(map[string]string)
	lableMap["cluster"] = c.Name
	lableMap["cluster_id"] = cast.ToString(c.Id)
	lableMap["cluster_type"] = c.Type.String()
	lableMap["region"] = c.Region
	lableMap["nodegroup"] = nodeGroup.Name
	lableMap["nodegroup_type"] = nodeGroup.Type.String()
	lablebytes, _ := json.Marshal(lableMap)
	return string(lablebytes)
}

func (uc *ClusterUsecase) GetCurrentCluster(ctx context.Context) (*Cluster, error) {
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) NodeGroupIncreaseSize(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup, size int32) error {
	for i := 0; i < int(size); i++ {
		node := &Node{
			Name:        fmt.Sprintf("%s-%s", cluster.Name, uuid.New().String()),
			Role:        NodeRole_WORKER,
			Status:      NodeStatus_NODE_CREATING,
			ClusterId:   cluster.Id,
			NodeGroupId: nodeGroup.Id,
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}
	return uc.Save(ctx, cluster)
}

func (uc *ClusterUsecase) DeleteNodes(ctx context.Context, cluster *Cluster, nodes []*Node) error {
	for _, node := range nodes {
		for i, n := range cluster.Nodes {
			if n.Id == node.Id {
				cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
				break
			}
		}
	}
	return uc.Save(ctx, cluster)
}

func (uc *ClusterUsecase) NodeGroupTemplateNodeInfo(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup) (*Node, error) {
	return &Node{
		Name:        fmt.Sprintf("%s-%s", cluster.Name, uuid.New().String()),
		Role:        NodeRole_WORKER,
		Status:      NodeStatus_NODE_CREATING,
		ClusterId:   cluster.Id,
		NodeGroupId: nodeGroup.Id,
		Labels:      cluster.generateNodeLables(nodeGroup),
	}, nil
}

func (uc *ClusterUsecase) Cleanup(ctx context.Context) error {
	return nil
}

func (uc *ClusterUsecase) Refresh(ctx context.Context) error {
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return err
	}
	cluster, err = uc.clusterData.GetByName(ctx, cluster.Name)
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
