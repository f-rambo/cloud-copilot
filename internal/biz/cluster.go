package biz

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/ansible"
	"github.com/f-rambo/ocean/pkg/kubeclient"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	ClusterRoleMaster = "master"
	ClusterRoleWorker = "worker"
	ClusterRoleEdge   = "edge"
)

type ClusterRepo interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	List(context.Context) ([]*Cluster, error)
	Delete(context.Context, int64) error
}

type ClusterUsecase struct {
	server *conf.Server
	repo   ClusterRepo
	log    *log.Helper
}

func NewClusterUseCase(server *conf.Server, repo ClusterRepo, logger log.Logger) *ClusterUsecase {
	return &ClusterUsecase{server: server, repo: repo, log: log.NewHelper(logger)}
}

func (uc *ClusterUsecase) CurrentCluster(ctx context.Context) (*Cluster, error) {
	clientSet, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return nil, err
	}
	versionInfo, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	serverAddress := clientSet.Discovery().RESTClient().Get().URL().Host
	cluster := &Cluster{Name: uc.server.Name, ServerVersion: versionInfo.String(), ApiServerAddress: serverAddress}
	err = uc.getNodes(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) getNodes(ctx context.Context, cluster *Cluster) error {
	clientSet, err := kubeclient.GetKubeClientSet()
	if err != nil {
		return err
	}
	nodeList, err := clientSet.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
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
		cluster.Nodes = append(cluster.Nodes, &Node{
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
	return nil
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

// param cluster: cluster to generate inventory file
// result: inventory file path
func (uc *ClusterUsecase) getInventory(ctx context.Context, cluster *Cluster) (string, error) {
	servers := make([]ansible.Server, 0)
	for _, node := range cluster.Nodes {
		servers = append(servers, ansible.Server{
			ID:       node.Name,
			Ip:       node.ExternalIP,
			Username: node.User,
			Role:     node.Role,
		})
	}
	ansibleInventory := ansible.GenerateInventoryFile(servers)
	file, err := utils.NewFile("./", "inventory.ini", false)
	if err != nil {
		return "", err
	}
	defer func() {
		if file == nil {
			return
		}
		err := file.Close()
		if err != nil {
			uc.log.Errorf("close file error: %v", err)
		}
	}()
	err = file.Write([]byte(ansibleInventory))
	if err != nil {
		return "", err
	}
	return file.GetFileName(), nil
}

// 安装集群
func (uc *ClusterUsecase) SetUpCluster(ctx context.Context, cluster *Cluster) error {
	return nil
}

// 添加节点
func (uc *ClusterUsecase) AddNode(ctx context.Context, cluster *Cluster, node *Node) error {
	return nil
}

// 删除节点
func (uc *ClusterUsecase) DeleteNode(ctx context.Context, cluster *Cluster, node *Node) error {
	return nil
}

// 卸载集群
func (uc *ClusterUsecase) UninstallCluster(ctx context.Context, cluster *Cluster) error {
	return nil
}
