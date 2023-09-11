package biz

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Cluster struct {
	ID       int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name     string `json:"cluster_name" gorm:"column:cluster_name; default:''; NOT NULL"`
	Nodes    []Node `json:"nodes" gorm:"-"`
	Deployed bool   `json:"deployed" gorm:"column:deployed;"`
	ClusterConfig
	ClusterSemaphore
	gorm.Model
}

type ClusterConfig struct {
	Config    datatypes.JSON `json:"-" gorm:"column:config; type:json"`
	Addons    datatypes.JSON `json:"-" gorm:"column:addons; type:json"`
	ConfigStr string         `json:"config_str" gorm:"column:-;"`
	AddonsStr string         `json:"Addons_str" gorm:"column:-;"`
}

type ClusterSemaphore struct {
	SemaphoreID int               `json:"semaphore_id" gorm:"column:semaphore_id; default:0; NOT NULL"`
	KeyID       int               `json:"key_id" gorm:"column:key_id; default:0; NOT NULL"`
	RepoID      int               `json:"repo_id" gorm:"column:repo_id; default:0; NOT NULL"`
	EnvID       int               `json:"env_id" gorm:"column:env_id; default:0; NOT NULL"`
	InventoryID int               `json:"inventory_id" gorm:"column:inventory_id; default:0; NOT NULL"`
	TemplateIDs datatypes.JSONMap `json:"template_ids" gorm:"column:template_ids; type:json"`
	TaskIDs     datatypes.JSONMap `json:"task_ids" gorm:"column:task_ids; type:json"`
}

type Node struct {
	ID           int      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string   `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Host         string   `json:"host" gorm:"column:host; default:''; NOT NULL"`
	User         string   `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Password     string   `json:"password" gorm:"column:password; default:''; NOT NULL"`
	SudoPassword string   `json:"sudo_password" gorm:"column:sudo_password; default:''; NOT NULL"`
	Role         []string `json:"role" gorm:"-"`                                            // master worker edge
	RoleJson     string   `json:"role_json" gorm:"column:role; default:''; NOT NULL"`       // gorm redundancy
	ClusterID    int      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"` // gorm redundancy
	gorm.Model
}

func (c *Cluster) SetSemaphoreID(id int) {
	c.SemaphoreID = id
}

func (c *Cluster) SetKeyID(id int) {
	c.KeyID = id
}

func (c *Cluster) SetRepoID(id int) {
	c.RepoID = id
}

func (c *Cluster) SetEnvID(id int) {
	c.EnvID = id
}

func (c *Cluster) SetInventoryID(id int) {
	c.InventoryID = id
}

func (c *Cluster) SetTemplateIDs(key string, val interface{}) {
	if c.TemplateIDs == nil {
		c.TemplateIDs = make(datatypes.JSONMap)
	}
	c.TemplateIDs[key] = val
}

func (c *Cluster) SetTaskID(key string, val interface{}) {
	if c.TaskIDs == nil {
		c.TaskIDs = make(datatypes.JSONMap)
	}
	c.TaskIDs[key] = val
}

func (c *Cluster) Merge(cluster *Cluster) {
	if cluster == nil {
		return
	}
	c.Name = cluster.Name
	c.SemaphoreID = cluster.SemaphoreID
	c.KeyID = cluster.KeyID
	c.RepoID = cluster.RepoID
	c.EnvID = cluster.EnvID
	c.InventoryID = cluster.InventoryID
	c.TemplateIDs = cluster.TemplateIDs
	c.CreatedAt = cluster.CreatedAt
	c.Deployed = false
}

type ClusterRepo interface {
	SaveCluster(ctx context.Context, cluster *Cluster) error
	GetClusters(ctx context.Context) ([]*Cluster, error)
	GetCluster(ctx context.Context, id int) (*Cluster, error)
	GetClusterByName(ctx context.Context, clusterName string) (*Cluster, error)
	DeleteCluster(ctx context.Context, cluster *Cluster) error
	ClusterInit(ctx context.Context, cluster *Cluster) error
	DeployCluster(ctx context.Context, cluster *Cluster) error
	UndeployCluster(ctx context.Context, cluster *Cluster) error
	AddNode(ctx context.Context, cluster *Cluster) error
	RemoveNode(ctx context.Context, cluster *Cluster, nodes []Node) error
	GetDefaultCluster(ctx context.Context) (*Cluster, error)
}

type ClusterUsecase struct {
	repo ClusterRepo
	log  *log.Helper
}

func NewClusterUseCase(repo ClusterRepo, logger log.Logger) *ClusterUsecase {
	return &ClusterUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (c *ClusterUsecase) Save(ctx context.Context, cluster *Cluster) error {
	if cluster.ID == 0 {
		clusterName, err := c.repo.GetClusterByName(ctx, cluster.Name)
		if err != nil {
			return err
		}
		if clusterName.ID != 0 {
			return errors.New("cluster name have already existed")
		}
	}
	currentCluster, err := c.repo.GetCluster(ctx, cluster.ID)
	if err != nil {
		return err
	}
	cluster.Merge(currentCluster)
	return c.repo.SaveCluster(ctx, cluster)
}

func (c *ClusterUsecase) Get(ctx context.Context) ([]*Cluster, error) {
	clusters, err := c.repo.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	if len(clusters) != 0 {
		return clusters, nil
	}
	// 默认数据
	clsuter, err := c.repo.GetDefaultCluster(ctx)
	if err != nil {
		return nil, err
	}
	return []*Cluster{clsuter}, nil
}

func (c *ClusterUsecase) Delete(ctx context.Context, clusterID int) error {
	cluster, err := c.repo.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}
	err = c.repo.UndeployCluster(ctx, cluster)
	if err != nil {
		return err
	}
	return c.repo.DeleteCluster(ctx, cluster)
}

func (c *ClusterUsecase) Apply(ctx context.Context, clusterName string) error {
	cluster, err := c.repo.GetClusterByName(ctx, clusterName)
	if err != nil {
		return err
	}
	if cluster.Deployed {
		return errors.New("cluster is applyed")
	}
	currentCluster, err := c.repo.GetCluster(ctx, cluster.ID)
	if err != nil {
		return err
	}
	if currentCluster == nil || !currentCluster.Deployed {
		err = c.repo.ClusterInit(ctx, cluster)
		if err != nil {
			return err
		}
		err = c.repo.DeployCluster(ctx, cluster)
		if err != nil {
			return err
		}
	}
	newNodes, rmNodes := c.getDiffNodes(currentCluster, cluster)
	if len(newNodes) != 0 {
		err = c.repo.ClusterInit(ctx, cluster)
		if err != nil {
			return err
		}
		err = c.repo.AddNode(ctx, cluster)
		if err != nil {
			return err
		}
	}
	if len(rmNodes) != 0 {
		err = c.repo.RemoveNode(ctx, cluster, rmNodes)
		if err != nil {
			return err
		}
	}
	cluster.Deployed = true
	return c.repo.SaveCluster(ctx, cluster)
}

func (c *ClusterUsecase) getDiffNodes(currentCluster, cluster *Cluster) (newNodes, removeNodes []Node) {
	newNodes = make([]Node, 0)
	removeNodes = make([]Node, 0)
	if currentCluster == nil || cluster == nil {
		return newNodes, removeNodes
	}
	for _, node := range cluster.Nodes {
		if node.ID == 0 {
			newNodes = append(newNodes, node)
			continue
		}
	}
	for _, node := range currentCluster.Nodes {
		if node.ID == 0 {
			continue
		}
		isExist := false
		for _, v := range cluster.Nodes {
			if v.ID == node.ID {
				isExist = true
				break
			}
		}
		if !isExist {
			removeNodes = append(removeNodes, node)
		}
	}
	return newNodes, removeNodes
}
