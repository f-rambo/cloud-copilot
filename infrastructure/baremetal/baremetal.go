package baremetal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v3"
)

var ARCH_MAP = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

var ArchMap = map[string]biz.NodeArchType{
	"x86_64":  biz.NodeArchType_AMD64,
	"aarch64": biz.NodeArchType_ARM64,
}

var GPUSpecMap = map[string]biz.NodeGPUSpec{
	"nvidia-a10":  biz.NodeGPUSpec_NVIDIA_A10,
	"nvidia-v100": biz.NodeGPUSpec_NVIDIA_V100,
	"nvidia-t4":   biz.NodeGPUSpec_NVIDIA_T4,
	"nvidia-p100": biz.NodeGPUSpec_NVIDIA_P100,
	"nvidia-p4":   biz.NodeGPUSpec_NVIDIA_P4,
}

var (
	Resource string = "resource"
	Shell    string = "shell"

	NodeInitShell   string = "nodeinit.sh"
	ComponentShell  string = "component.sh"
	SystemInfoShell string = "systeminfo.sh"
	ClusterInstall  string = "clusterinstall.sh"

	ClusterConfiguration string = "cluster-config.yaml"
	Install              string = "install.yaml"

	ClusterInitAction string = "init"
	ClusterJoinAction string = "join"
	ClusterController string = "controller"

	ResourcePackageUrl = "https://github.com/f-rambo/infrastructure/releases/download/v0.0.1/resource-v0.0.1.tar.gz"
)

type Baremetal struct {
	log *log.Helper
}

func NewBaremetal(logger log.Logger) *Baremetal {
	return &Baremetal{log: log.NewHelper(logger)}
}

func (b *Baremetal) getClusterNodeRemoteBash(cluster *biz.Cluster, node *biz.Node) *utils.RemoteBash {
	return utils.NewRemoteBash(utils.Server{
		Name:       node.Name,
		Host:       node.Ip,
		User:       node.User,
		Port:       22,
		PrivateKey: cluster.PrivateKey,
	}, Shell, b.log)
}

func (b *Baremetal) nodeInstallInit(cluster *biz.Cluster, node *biz.Node) error {
	remoteBash := b.getClusterNodeRemoteBash(cluster, node)
	err := remoteBash.ExecShellLogging(NodeInitShell, node.Name)
	if err != nil {
		return err
	}
	userHomePath, err := remoteBash.GetUserHome()
	if err != nil {
		return err
	}
	err = remoteBash.ExecShellLogging(ComponentShell, filepath.Join(userHomePath, Resource), cluster.ImageRepo, cluster.KuberentesVersion)
	if err != nil {
		return err
	}
	return nil
}

func (b *Baremetal) migrateResources(cluster *biz.Cluster, node *biz.Node) error {
	remoteBash := b.getClusterNodeRemoteBash(cluster, node)
	userHomePath, err := remoteBash.GetUserHome()
	if err != nil {
		return err
	}
	resourcePackageDir := fmt.Sprintf("ls %s | wc -l", filepath.Join(userHomePath, Resource))
	fileNumber, err := remoteBash.Run(resourcePackageDir)
	if err != nil {
		return err
	}
	if cast.ToInt(strings.TrimSpace(fileNumber)) > 0 {
		return nil
	}
	tarFilename, err := utils.DownloadFile(ResourcePackageUrl)
	if err != nil {
		return err
	}
	remoteTarfile := fmt.Sprintf("/tmp/%s", tarFilename)
	fileNumber, err = remoteBash.Run("ls", remoteTarfile, "| wc -l")
	if err != nil {
		return err
	}
	if cast.ToInt(strings.TrimSpace(fileNumber)) == 0 {
		err = remoteBash.SftpFile(tarFilename, remoteTarfile)
		if err != nil {
			return err
		}
	}
	err = remoteBash.RunWithLogging("tar", "-C", userHomePath, "-zxvf", remoteTarfile)
	if err != nil {
		return err
	}
	return nil
}

func (b *Baremetal) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.SetLimit(10)
	nodeInforMaps := make([]map[string]string, 0)
	lock := new(sync.Mutex)
	for _, v := range cluster.Nodes {
		node := v
		errGroup.Go(func() error {
			nodeInfoMap := make(map[string]string)
			remoteBash := utils.NewRemoteBash(utils.Server{
				Name:       node.Name,
				Host:       node.Ip,
				User:       node.User,
				Port:       22,
				PrivateKey: cluster.PrivateKey,
			}, Shell, b.log)
			systemInfoOutput, err := remoteBash.ExecShell(SystemInfoShell)
			if err != nil {
				b.log.Errorf("node %s connection refused", node.Ip)
				return nil
			}
			systemInfoMap := make(map[string]any)
			if err := json.Unmarshal([]byte(systemInfoOutput), &systemInfoMap); err != nil {
				return err
			}
			for key, val := range systemInfoMap {
				nodeInfoMap[key] = cast.ToString(val)
			}
			lock.Lock()
			nodeInforMaps = append(nodeInforMaps, nodeInfoMap)
			lock.Unlock()
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	nodeGroupMaps := make(map[string][]*biz.Node)
	for _, m := range nodeInforMaps {
		nodegroup := &biz.NodeGroup{}
		node := &biz.Node{}
		for key, val := range m {
			switch key {
			case "os":
				nodegroup.Os = val
			case "arch":
				arch, ok := ArchMap[val]
				if !ok {
					arch = biz.NodeArchType_UNSPECIFIED
				}
				nodegroup.Arch = arch
			case "mem":
				nodegroup.Memory = cast.ToInt32(val)
			case "cpu":
				nodegroup.Cpu = cast.ToInt32(val)
			case "gpu":
				nodegroup.Gpu = cast.ToInt32(val)
			case "gpu_info":
				gpuSpec, ok := GPUSpecMap[val]
				if !ok {
					gpuSpec = biz.NodeGPUSpec_UNSPECIFIED
				}
				nodegroup.GpuSpec = gpuSpec
			case "disk":
				node.SystemDiskSize = cast.ToInt32(val)
			case "ip":
				node.Ip = cast.ToString(val)
			}
		}
		nodeGroupMaps[cluster.EncodeNodeGroup(nodegroup)] = append(nodeGroupMaps[cluster.EncodeNodeGroup(nodegroup)], node)
	}

	// Init node group and node
	cluster.NodeGroups = make([]*biz.NodeGroup, 0)
	cluster.Nodes = make([]*biz.Node, 0)
	for nodeGroupEncodeKey, nodes := range nodeGroupMaps {
		nodeGroupExits := false
		nodeGrpupId := ""
		for _, ng := range cluster.NodeGroups {
			if cluster.EncodeNodeGroup(ng) == nodeGroupEncodeKey {
				nodeGrpupId = ng.Id
				nodeGroupExits = true
				break
			}
		}
		if nodeGroupExits {
			for _, node := range nodes {
				nodeExits := false
				for _, n := range cluster.Nodes {
					if n.Ip == node.Ip {
						nodeExits = true
						break
					}
				}
				if !nodeExits {
					node.ClusterId = cluster.Id
					node.NodeGroupId = nodeGrpupId
					node.User = "root"
					node.Name = node.Ip
					cluster.Nodes = append(cluster.Nodes, node)
				}
			}
			continue
		}
		nodegroup := cluster.DecodeNodeGroup(nodeGroupEncodeKey)
		nodegroup.Id = uuid.NewString()
		for _, node := range nodes {
			node.ClusterId = cluster.Id
			node.NodeGroupId = nodegroup.Id
			node.User = "root"
			node.Name = node.Ip
		}
		cluster.NodeGroups = append(cluster.NodeGroups, nodegroup)
		cluster.Nodes = append(cluster.Nodes, nodes...)
	}
	return nil
}

func (b *Baremetal) ApplyServices(ctx context.Context, cluster *biz.Cluster) error {
	var node *biz.Node
	for _, v := range cluster.Nodes {
		if v.Role == biz.NodeRole_MASTER {
			node = v
			break
		}
	}
	if node == nil {
		return errors.New("no master node found")
	}
	if cluster.Provider.IsCloud() {
		slb := cluster.GetSingleCloudResource(biz.ResourceType_LOAD_BALANCER)
		node.Ip = slb.Value
	}
	remoteBash := b.getClusterNodeRemoteBash(cluster, node)
	userHomePath, err := remoteBash.GetUserHome()
	if err != nil {
		return err
	}
	arch, err := remoteBash.Run("uname", "-m")
	if err != nil {
		return err
	}
	arch = strings.TrimSpace(strings.ToLower(arch))
	archMapVal, ok := ARCH_MAP[arch]
	if ok {
		arch = archMapVal
	}
	kubeCtlPath := filepath.Join(userHomePath, Resource, arch, "kubernetes", cluster.KuberentesVersion, "kubectl")
	err = remoteBash.RunWithLogging("install -m 755", kubeCtlPath, "/usr/local/bin/kubectl")
	if err != nil {
		return err
	}
	installfile, err := utils.TransferredMeaning(cluster, Install)
	if err != nil {
		return err
	}
	err = remoteBash.SftpFile(installfile, filepath.Join(userHomePath, Install))
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("kubectl apply -f", filepath.Join(userHomePath, Install))
	if err != nil {
		return err
	}
	return nil
}

func (b *Baremetal) Install(ctx context.Context, cluster *biz.Cluster) error {
	var node *biz.Node
	for _, v := range cluster.Nodes {
		if v.Role == biz.NodeRole_MASTER {
			node = v
			break
		}
	}
	if node == nil {
		return errors.New("no master node found")
	}
	if cluster.Provider.IsCloud() {
		slb := cluster.GetSingleCloudResource(biz.ResourceType_LOAD_BALANCER)
		node.Ip = slb.Value
	}
	remoteBash := b.getClusterNodeRemoteBash(cluster, node)
	err := b.migrateResources(cluster, node)
	if err != nil {
		return err
	}
	err = b.nodeInstallInit(cluster, node)
	if err != nil {
		return err
	}
	cluster.Config, err = utils.TransferredMeaningString(cluster, ClusterConfiguration)
	if err != nil {
		return err
	}
	clusterYaml, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}
	err = remoteBash.ExecShellLogging(ClusterInstall, ClusterInitAction, string(clusterYaml))
	if err != nil {
		return err
	}
	return nil
}

func (b *Baremetal) UnInstall(cluster *biz.Cluster) error {
	for _, node := range cluster.Nodes {
		err := b.uninstallNode(cluster, node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Baremetal) HandlerNodes(cluster *biz.Cluster) error {
	for _, node := range cluster.Nodes {
		if node.Status == biz.NodeStatus_NODE_CREATING {
			err := b.joinCluster(cluster, node)
			if err != nil {
				return err
			}
		}
		if node.Status == biz.NodeStatus_NODE_DELETING {
			err := b.uninstallNode(cluster, node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Baremetal) joinCluster(cluster *biz.Cluster, node *biz.Node) error {
	err := b.migrateResources(cluster, node)
	if err != nil {
		return err
	}
	err = b.nodeInstallInit(cluster, node)
	if err != nil {
		return err
	}
	clusterYaml, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}
	if node.Role != biz.NodeRole_MASTER {
		err = b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(ClusterInstall, ClusterJoinAction, string(clusterYaml))
		if err != nil {
			return err
		}
		return nil
	}
	err = b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(ClusterInstall, ClusterJoinAction, string(clusterYaml), ClusterController)
	if err != nil {
		return err
	}
	return nil
}

func (b *Baremetal) uninstallNode(cluster *biz.Cluster, node *biz.Node) error {
	remoteBash := utils.NewRemoteBash(utils.Server{
		Name:       node.Name,
		Host:       node.Ip,
		User:       node.User,
		Port:       22,
		PrivateKey: cluster.PrivateKey,
	}, Shell, b.log)
	err := remoteBash.RunWithLogging("sudo kubeadm reset --force")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("sudo rm -rf $HOME/.kube && rm -rf /etc/kubernetes && rm -rf /etc/cni")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("sudo systemctl stop containerd && systemctl disable containerd && rm -rf /var/lib/containerd")
	if err != nil {
		return err
	}
	err = remoteBash.RunWithLogging("sudo systemctl stop kubelet && systemctl disable kubelet && rm -rf /var/lib/kubelet")
	if err != nil {
		return err
	}
	return nil
}
