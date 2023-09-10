package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Cluster struct {
	ID              int               `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	ClusterName     string            `json:"cluster_name" gorm:"column:cluster_name; default:''; NOT NULL"`
	Nodes           []Node            `json:"nodes" gorm:"-"`
	Config          datatypes.JSON    `json:"config" gorm:"column:config; type:json"`
	Addons          datatypes.JSON    `json:"addons" gorm:"column:addons; type:json"`
	User            string            `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Password        string            `json:"password" gorm:"column:password; default:''; NOT NULL"`
	SudoPassword    string            `json:"sudo_password" gorm:"column:sudo_password; default:''; NOT NULL"`
	SemaphoreID     int               `json:"semaphore_id" gorm:"column:semaphore_id; default:0; NOT NULL"`
	RootUserKeyID   int               `json:"root_user_key_id" gorm:"column:root_user_key_id; default:0; NOT NULL"`
	NormalUserKeyID int               `json:"normal_user_key_id" gorm:"column:normal_user_key_id; default:0; NOT NULL"`
	RepoID          int               `json:"repo_id" gorm:"column:repo_id; default:0; NOT NULL"`
	EnvID           int               `json:"env_id" gorm:"column:env_id; default:0; NOT NULL"`
	InventoryID     int               `json:"inventory_id" gorm:"column:inventory_id; default:0; NOT NULL"`
	TemplateIDs     datatypes.JSONMap `json:"template_ids" gorm:"column:template_ids; type:json"`
	gorm.Model
}

type Node struct {
	ID        int      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string   `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Host      string   `json:"host" gorm:"column:host; default:''; NOT NULL"`
	Role      []string `json:"role" gorm:"-"`                                            // master worker edge
	RoleJson  string   `json:"role_json" gorm:"column:role; default:''; NOT NULL"`       // gorm redundancy
	ClusterID int      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"` // gorm redundancy
	gorm.Model
}

func (c *Cluster) SetSemaphoreID(id int) {
	c.SemaphoreID = id
}

func (c *Cluster) SetRootUserKeyID(id int) {
	c.RootUserKeyID = id
}

func (c *Cluster) SetNormalUserKeyID(id int) {
	c.NormalUserKeyID = id
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

func (c *Cluster) Merge(cluster *Cluster) {
	if cluster == nil {
		return
	}
	c.ClusterName = cluster.ClusterName
	c.SemaphoreID = cluster.SemaphoreID
	c.RootUserKeyID = cluster.RootUserKeyID
	c.NormalUserKeyID = cluster.NormalUserKeyID
	c.RepoID = cluster.RepoID
	c.EnvID = cluster.EnvID
	c.InventoryID = cluster.InventoryID
	c.TemplateIDs = cluster.TemplateIDs
	c.CreatedAt = cluster.CreatedAt
}

type ClusterRepo interface {
	SaveCluster(ctx context.Context, cluster *Cluster) error
	GetClusters(ctx context.Context) ([]*Cluster, error)
	GetCluster(ctx context.Context, id int) (*Cluster, error)
	DeleteCluster(ctx context.Context, cluster *Cluster) error
	ClusterInit(ctx context.Context, cluster *Cluster) error
	DeployCluster(ctx context.Context, cluster *Cluster) error
	UndeployCluster(ctx context.Context, cluster *Cluster) error
	AddNode(ctx context.Context, cluster *Cluster) error
	RemoveNode(ctx context.Context, cluster *Cluster, nodes []*Node) error
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
	historyCluster, err := c.repo.GetCluster(ctx, cluster.ID)
	if err != nil {
		return err
	}
	cluster.Merge(historyCluster)
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
	return c.repo.DeleteCluster(ctx, cluster)
}
