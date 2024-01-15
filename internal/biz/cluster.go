package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

type Cluster struct {
	ID               int64   `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name             string  `json:"name" gorm:"column:name; default:''; NOT NULL"`
	ServerVersion    string  `json:"server_version" gorm:"column:server_version; default:''; NOT NULL"`
	ApiServerAddress string  `json:"api_server_address" gorm:"column:api_server_address; default:''; NOT NULL"`
	Config           string  `json:"config" gorm:"column:config; default:''; NOT NULL;"`
	Addons           string  `json:"addons" gorm:"column:addons; default:''; NOT NULL;"`
	Nodes            []*Node `json:"nodes" gorm:"-"`
	gorm.Model
}

type Node struct {
	ID           int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name         string `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Labels       string `json:"labels" gorm:"column:labels; default:''; NOT NULL"`
	Annotations  string `json:"annotations" gorm:"column:annotations; default:''; NOT NULL"`
	OSImage      string `json:"os_image" gorm:"column:os_image; default:''; NOT NULL"`
	Kernel       string `json:"kernel" gorm:"column:kernel; default:''; NOT NULL"`
	Container    string `json:"container" gorm:"column:container; default:''; NOT NULL"`
	Kubelet      string `json:"kubelet" gorm:"column:kubelet; default:''; NOT NULL"`
	KubeProxy    string `json:"kube_proxy" gorm:"column:kube_proxy; default:''; NOT NULL"`
	InternalIP   string `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	ExternalIP   string `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	User         string `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Password     string `json:"password" gorm:"column:password; default:''; NOT NULL"`
	SudoPassword string `json:"sudo_password" gorm:"column:sudo_password; default:''; NOT NULL"`
	Role         string `json:"role" gorm:"column:role; default:''; NOT NULL;"` // master worker edge
	ClusterID    int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	gorm.Model
}

type ClusterRepo interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	List(context.Context) ([]*Cluster, error)
	Delete(context.Context, int64) error
	ClusterClient(ctx context.Context, clusterID int64) (*kubernetes.Clientset, error)
}

type ClusterUsecase struct {
	repo ClusterRepo
	log  *log.Helper
}

func NewClusterUseCase(repo ClusterRepo, logger log.Logger) *ClusterUsecase {
	return &ClusterUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ClusterUsecase) Save(ctx context.Context, cluster *Cluster) error {
	return uc.repo.Save(ctx, cluster)
}

func (uc *ClusterUsecase) Get(ctx context.Context, id int64) (*Cluster, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *ClusterUsecase) List(ctx context.Context) ([]*Cluster, error) {
	return uc.repo.List(ctx)
}

func (uc *ClusterUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
