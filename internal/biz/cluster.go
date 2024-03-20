package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/ansible"
	"github.com/f-rambo/ocean/pkg/kubeclient"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
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
	AddonsConfig     string  `json:"addons_config" gorm:"column:addons_config; default:''; NOT NULL;"`
	State            string  `json:"state" gorm:"column:state; default:''; NOT NULL;"`
	Nodes            []*Node `json:"nodes" gorm:"-"`
	Logs             string  `json:"logs" gorm:"-"` // logs data from localfile
	gorm.Model
}

func (c *Cluster) GetNode(nodeId int64) *Node {
	for _, node := range c.Nodes {
		if node.ID == nodeId {
			return node
		}
	}
	return nil
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
	State        string `json:"state" gorm:"column:state; default:''; NOT NULL"`
	ClusterID    int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	gorm.Model
}

const (
	ClusterStateInit     = "init"
	ClusterStateChecked  = "checked"
	ClusterStateNotReady = "not ready"
	ClusterStateRunning  = "running"
	ClusterStateFailed   = "failed"
	ClusterStateFinished = "finished"
)

const (
	NodeSateInit      = "init"
	NodeStateRunning  = "running"
	NodeStateFailed   = "failed"
	NodeStateFinished = "finished"
)

const (
	ClusterRoleMaster = "master"
	ClusterRoleWorker = "worker"
	ClusterRoleEdge   = "edge"
)

type ClusterRepo interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	List(context.Context, *Cluster) ([]*Cluster, error)
	Delete(context.Context, int64) error
	ReadClusterLog(cluster *Cluster) error
	WriteClusterLog(cluster *Cluster) error
}

type ClusterUsecase struct {
	c    *conf.Bootstrap
	repo ClusterRepo
	log  *log.Helper
}

func NewClusterUseCase(c *conf.Bootstrap, repo ClusterRepo, logger log.Logger) *ClusterUsecase {
	return &ClusterUsecase{
		c:    c,
		repo: repo,
		log:  log.NewHelper(logger),
	}
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
	cserver := uc.c.GetOceanServer()
	cluster := &Cluster{Name: cserver.Name, ServerVersion: versionInfo.String(), ApiServerAddress: serverAddress}
	err = uc.getNodes(cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) getNodes(cluster *Cluster) error {
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
	if cluster.ID != 0 {
		return uc.repo.Save(ctx, cluster)
	}
	clusters, err := uc.repo.List(ctx, &Cluster{Name: cluster.Name})
	if err != nil {
		return err
	}
	if len(clusters) > 0 {
		return errors.New("cluster name already exists")
	}
	cluster.State = ClusterStateInit
	return uc.repo.Save(ctx, cluster)
}

func (uc *ClusterUsecase) Get(ctx context.Context, id int64) (*Cluster, error) {
	cluster, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	err = uc.repo.ReadClusterLog(cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) List(ctx context.Context) ([]*Cluster, error) {
	return uc.repo.List(ctx, nil)
}

func (uc *ClusterUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *ClusterUsecase) DeleteNode(ctx context.Context, clusterID int64, nodeID int64) error {
	cluster, err := uc.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	for i, node := range cluster.Nodes {
		if node.ID == nodeID {
			cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
			break
		}
	}
	return uc.Save(ctx, cluster)
}

// 安装集群
func (uc *ClusterUsecase) SetUpCluster(ctx context.Context, clusterID int64) error {
	cluster, err := uc.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	execplayBookParam := newExecPlaybookParam().
		setCtx(context.TODO()).
		setCluster(cluster).
		setPlaybooks(kubespray.GetClusterPath()).
		setCmdRunDir(kubespray.GetPackagePath())
	go func() {
		err = uc.execPlaybook(execplayBookParam)
		if err != nil {
			uc.log.Errorf("setup cluster error: %v", err)
			return
		}
	}()
	return nil
}

// 卸载集群
func (uc *ClusterUsecase) UninstallCluster(ctx context.Context, clusterID int64) error {
	cluster, err := uc.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	execplayBookParam := newExecPlaybookParam().
		setCtx(context.TODO()).
		setCluster(cluster).
		setPlaybooks(kubespray.GetResetPath()).
		setCmdRunDir(kubespray.GetPackagePath())
	go func() {
		err = uc.execPlaybook(execplayBookParam)
		if err != nil {
			uc.log.Errorf("uninstall cluster error: %v", err)
			return
		}
	}()
	return nil
}

// 添加节点
func (uc *ClusterUsecase) AddNode(ctx context.Context, clusterID int64, nodeID int64) error {
	cluster, err := uc.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	node := cluster.GetNode(nodeID)
	if node == nil {
		return errors.New("node not found")
	}
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	execplayBookParam := newExecPlaybookParam().
		setCtx(context.TODO()).
		setCluster(cluster).
		setPlaybooks(kubespray.GetScalePath()).
		setCmdRunDir(kubespray.GetPackagePath())
	go func() {
		err = uc.execPlaybook(execplayBookParam)
		if err != nil {
			uc.log.Errorf("add nodes error: %v", err)
			return
		}
	}()
	return nil
}

// 移除节点
func (uc *ClusterUsecase) RemoveNode(ctx context.Context, clusterID int64, nodeID int64) error {
	cluster, err := uc.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	node := cluster.GetNode(nodeID)
	if node == nil {
		return errors.New("node not found")
	}
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	execplayBookParam := newExecPlaybookParam().
		setCtx(context.TODO()).
		setCluster(cluster).
		setPlaybooks(kubespray.GetRemoveNodePath()).
		setCmdRunDir(kubespray.GetPackagePath()).
		setEnv("node", node.Name)
	go func() {
		err = uc.execPlaybook(execplayBookParam)
		if err != nil {
			uc.log.Errorf("remove nodes error: %v", err)
			return
		}
	}()
	return nil
}

func (uc *ClusterUsecase) CheckConfig(ctx context.Context, clusterID int64) (*Cluster, error) {
	cluster, err := uc.repo.Get(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = uc.repo.Save(ctx, cluster)
		if err != nil {
			uc.log.Errorf("save cluster error: %v", err)
			return
		}
	}()
	checkFuncs := []func(ctx context.Context, cluster *Cluster) error{
		uc.ansibleCfgInit,
		uc.checkServerConfig,
		uc.checkClusterConfig,
		uc.checkClusterAddons,
	}
	for _, f := range checkFuncs {
		err = f(ctx, cluster)
		if err != nil {
			cluster.State = ClusterStateNotReady
			return cluster, err
		}
	}
	cluster.State = ClusterStateChecked
	return cluster, nil
}

func (uc *ClusterUsecase) ansibleCfgInit(ctx context.Context, _ *Cluster) error {
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	ansiblePkg := ansible.NewGoAnsiblePkg(uc.c)
	ansibleCfgContent, err := ansiblePkg.GenerateAnsibleCfg()
	if err != nil {
		return err
	}
	file, err := utils.NewFile(kubespray.GetPackagePath(), "ansible.cfg", true)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			uc.log.Errorf("ansibleCfgInit panic: %v", r)
		}
		err := file.Close()
		if err != nil {
			uc.log.Errorf("close file error: %v", err)
		}
	}()
	err = file.ClearFileContent()
	if err != nil {
		return err
	}
	err = file.Write([]byte(ansibleCfgContent))
	if err != nil {
		return err
	}
	return nil

}

func (uc *ClusterUsecase) checkServerConfig(ctx context.Context, cluster *Cluster) error {
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return err
	}
	file, err := utils.NewFile(kubespray.GetPackagePath(), "server-init.yml", true)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			uc.log.Errorf("checkServerConfig panic: %v", r)
		}
		err := file.Close()
		if err != nil {
			uc.log.Errorf("close file error: %v", err)
		}
	}()
	err = file.ClearFileContent()
	if err != nil {
		return err
	}
	ansiblePkg := ansible.NewGoAnsiblePkg(uc.c)
	serverinitPlaybook, err := ansiblePkg.GenerateServerInitPlaybook()
	if err != nil {
		return err
	}
	err = file.Write([]byte(serverinitPlaybook))
	if err != nil {
		return err
	}
	execPlayBookParam := newExecPlaybookParam().
		setCluster(cluster).
		setCtx(ctx).
		setPlaybooks(file.GetFileName()).
		setCmdRunDir(kubespray.GetPackagePath())
	return uc.execPlaybook(execPlayBookParam)
}

func (uc *ClusterUsecase) checkClusterConfig(ctx context.Context, cluster *Cluster) error {
	return nil
}

func (uc *ClusterUsecase) checkClusterAddons(ctx context.Context, cluter *Cluster) error {
	return nil
}

// param cluster: cluster to generate inventory file
// result: inventory file name
func (uc *ClusterUsecase) getInventory(cluster *Cluster) (string, error) {
	servers := make([]ansible.Server, 0)
	for _, node := range cluster.Nodes {
		servers = append(servers, ansible.Server{
			ID:       node.Name,
			Ip:       node.ExternalIP,
			Username: node.User,
			Role:     node.Role,
		})
	}
	cresource := uc.c.GetOceanResource()
	kubespray, err := ansible.NewKubespray(&cresource)
	if err != nil {
		return "", err
	}
	ansibleInventory := ansible.GenerateInventoryFile(servers)
	file, err := utils.NewFile(kubespray.GetPackagePath(), fmt.Sprintf("inventory-%d.ini", cluster.ID), true)
	if err != nil {
		return "", err
	}
	err = file.ClearFileContent()
	if err != nil {
		return "", err
	}
	defer func() {
		if r := recover(); r != nil {
			uc.log.Errorf("getInventory panic: %v", r)
		}
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

type execPlaybookParam struct {
	ctx       context.Context
	cluster   *Cluster
	env       map[string]string
	cmdRunDir string
	playbooks []string
}

func newExecPlaybookParam() *execPlaybookParam {
	return &execPlaybookParam{}
}

func (e execPlaybookParam) setCtx(ctx context.Context) *execPlaybookParam {
	e.ctx = ctx
	return &e
}

func (e execPlaybookParam) setCluster(cluster *Cluster) *execPlaybookParam {
	e.cluster = cluster
	return &e
}

func (e execPlaybookParam) setEnv(key, val string) *execPlaybookParam {
	if e.env == nil {
		e.env = make(map[string]string)
	}
	e.env[key] = val
	return &e
}

func (e execPlaybookParam) setCmdRunDir(cmdRunDir string) *execPlaybookParam {
	e.cmdRunDir = cmdRunDir
	return &e
}

func (e execPlaybookParam) setPlaybooks(playbooks ...string) *execPlaybookParam {
	e.playbooks = playbooks
	return &e
}

func (uc *ClusterUsecase) execPlaybook(param *execPlaybookParam) error {
	if param.cluster == nil {
		return errors.New("cluster is required")
	}
	if param.cmdRunDir == "" {
		return errors.New("cmdRunDir is required")
	}
	if len(param.playbooks) == 0 {
		return errors.New("playbooks is required")
	}
	if param.ctx == nil {
		param.ctx = context.Background()
	}
	inventoryFilePathName, err := uc.getInventory(param.cluster)
	if err != nil {
		return err
	}
	cresource := uc.c.GetOceanResource()
	g := new(errgroup.Group)
	ansibleLog := make(chan string, 100)
	ansibleObj := ansible.NewGoAnsiblePkg(uc.c).
		SetAnsiblePlaybookBinary(cresource.GetAnsibleCli()).
		SetLogChan(ansibleLog).
		SetInventoryFile(inventoryFilePathName).
		SetCmdRunDir(param.cmdRunDir).
		SetPlaybooks(param.playbooks).
		SetEnvMap(param.env)
	g.Go(func() error {
		defer func() {
			close(ansibleLog)
			err := recover()
			if err != nil {
				uc.log.Errorf("execPlaybook panic: %v", err)
			}
		}()
		err = ansibleObj.ExecPlayBooks(param.ctx)
		if err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				uc.log.Errorf("execPlaybook panic: %v", r)
			}
			if param.cluster.Logs != "" {
				err = uc.repo.WriteClusterLog(param.cluster)
				if err != nil {
					uc.log.Errorf("write cluster log error: %v", err)
					return
				}
				param.cluster.Logs = ""
			}
		}()
		for {
			select {
			case log, ok := <-ansibleLog:
				if !ok {
					return nil
				}
				param.cluster.Logs += log
			case <-time.After(10 * time.Second):
				err = uc.repo.WriteClusterLog(param.cluster)
				if err != nil {
					return err
				}
				param.cluster.Logs = ""
			}
		}
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
