package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type Cluster struct {
	ClusterName string `yaml:"cluster_name"`
	Nodes       []Node `yaml:"nodes"`
}

type Node struct {
	Name         string   `yaml:"name"`
	Host         string   `yaml:"host"`
	User         string   `yaml:"user"`
	Password     string   `yaml:"password"`
	SudoPassword string   `yaml:"sudo_password"`
	Role         []string `yaml:"role"`
}

type ClusterRepo interface {
	SaveCluster(context.Context, *Cluster) error
	GetCluster(context.Context) (*Cluster, error)
	SetUpClusterTool(context.Context, *Cluster) error
	DeployCluster(context.Context, *Cluster) error
	SetClusterAuth(context.Context, *Cluster) error
	SyncConfigCluster(context.Context) error
	DestroyCluster(context.Context, *Cluster) error
	AddNodes(context.Context, *Cluster) error
	RemoveNodes(context.Context, []string) error
	ClusterDataWatch(func(*Cluster, *Cluster) error) error
	GetClusterConfig(context.Context, *Cluster, string) ([]byte, error)
	SaveClusterConfig(context.Context, *Cluster, string, []byte) error
}

type ClusterUsecase struct {
	repo ClusterRepo
	log  *log.Helper
}

func NewClusterUseCase(repo ClusterRepo, logger log.Logger) *ClusterUsecase {
	cluster := &ClusterUsecase{repo: repo, log: log.NewHelper(logger)}
	go func() {
		for {
			err := cluster.repo.ClusterDataWatch(cluster.SupervisoryControl)
			if err != nil {
				cluster.log.Errorf("supervisory control failed: %v", err)
			}
			time.Sleep(time.Duration(10) * time.Second)
		}
	}()
	return cluster
}

func (c *ClusterUsecase) SupervisoryControl(old, new *Cluster) error {
	ctx := context.Background()
	if old == nil || new == nil {
		return nil
	}
	addNodes := make([]string, 0)
	rmNodes := make([]string, 0)
	for _, newnode := range new.Nodes {
		isExist := false
		for _, oldnode := range old.Nodes {
			if newnode.Name == oldnode.Name {
				isExist = true
				break
			}
		}
		if !isExist {
			addNodes = append(addNodes, newnode.Name)
		}
	}
	for _, oldnode := range old.Nodes {
		isExist := false
		for _, newnode := range new.Nodes {
			if oldnode.Name == newnode.Name {
				isExist = true
				break
			}
		}
		if !isExist {
			rmNodes = append(rmNodes, oldnode.Name)
		}
	}
	if len(rmNodes) > 0 {
		err := c.removeNodes(ctx, rmNodes)
		if err != nil {
			c.log.Errorf("remove nodes failed: %v", err)
		}
	}
	if len(addNodes) > 0 {
		err := c.addNodes(ctx)
		if err != nil {
			c.log.Errorf("add nodes failed: %v", err)
		}
	}
	return nil
}

func (c *ClusterUsecase) addNodes(ctx context.Context) error {
	clusterData, err := c.repo.GetCluster(ctx)
	if err != nil {
		return nil
	}
	return c.repo.AddNodes(ctx, clusterData)
}

func (c *ClusterUsecase) removeNodes(ctx context.Context, nodes []string) error {
	return c.repo.RemoveNodes(ctx, nodes)
}

func (c *ClusterUsecase) SaveCluster(ctx context.Context, cluster *Cluster) error {
	return c.repo.SaveCluster(ctx, cluster)
}

func (c *ClusterUsecase) GetCluster(ctx context.Context) (*Cluster, error) {
	return c.repo.GetCluster(ctx)
}

func (c *ClusterUsecase) SyncConfigCluster(ctx context.Context) error {
	return c.repo.SyncConfigCluster(ctx)
}

func (c *ClusterUsecase) SetClusterAuth(ctx context.Context) error {
	clusterData, err := c.repo.GetCluster(ctx)
	if err != nil {
		return nil
	}
	return c.repo.SetClusterAuth(ctx, clusterData)
}

func (c *ClusterUsecase) SetUpClusterTool(ctx context.Context) error {
	clusterData, err := c.repo.GetCluster(ctx)
	if err != nil {
		return nil
	}
	return c.repo.SetUpClusterTool(ctx, clusterData)
}

func (c *ClusterUsecase) DeployCluster(ctx context.Context) error {
	clusterData, err := c.repo.GetCluster(ctx)
	if err != nil {
		return nil
	}
	go func() {
		err = c.repo.DeployCluster(ctx, clusterData)
		if err != nil {
			c.log.Errorf("deploy cluster failed: %v", err)
		}
	}()
	return nil
}

func (c *ClusterUsecase) DestroyCluster(ctx context.Context) error {
	clusterData, err := c.repo.GetCluster(ctx)
	if err != nil {
		return nil
	}
	go func() {
		err = c.repo.DestroyCluster(ctx, clusterData)
		if err != nil {
			c.log.Errorf("destroy cluster failed: %v", err)
		}
	}()
	return nil
}

func (c *ClusterUsecase) GetClusterConfig(ctx context.Context, module string) ([]byte, error) {
	cluster, err := c.GetCluster(ctx)
	if err != nil {
		return nil, err
	}
	return c.repo.GetClusterConfig(ctx, cluster, module)
}

func (c *ClusterUsecase) SaveClusterConfig(ctx context.Context, module string, data []byte) error {
	cluster, err := c.GetCluster(ctx)
	if err != nil {
		return err
	}
	return c.repo.SaveClusterConfig(ctx, cluster, module, data)
}
