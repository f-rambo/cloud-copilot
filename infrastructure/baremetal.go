package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v3"
)

type Baremetal struct {
	c   *conf.Bootstrap
	log *log.Helper
}

func NewBaremetal(c *conf.Bootstrap, logger log.Logger) *Baremetal {
	return &Baremetal{c: c, log: log.NewHelper(logger)}
}

func (b *Baremetal) getClusterNodeRemoteBash(cluster *biz.Cluster, node *biz.Node) *utils.RemoteBash {
	return utils.NewRemoteBash(utils.Server{
		Name:       node.Name,
		Host:       node.Ip,
		User:       node.User,
		Port:       defaultSHHPort,
		PrivateKey: cluster.PrivateKey,
	}, b.c.Infrastructure.Shell, b.log)
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
	k8sImageRepo := getDefaultKuberentesImageRepo()
	if cluster.Provider == biz.ClusterProvider_AliCloud {
		k8sImageRepo = getAliyunKuberentesImageRepo()
	}
	err = remoteBash.ExecShellLogging(ComponentShell,
		filepath.Join(userHomePath, b.c.Infrastructure.Resource), k8sImageRepo, getKubernetesVersion())
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
	remoteResroucePath := filepath.Join(userHomePath, b.c.Infrastructure.Resource)
	fileNumber, err := remoteBash.Run(fmt.Sprintf("ls %s | wc -l", remoteResroucePath))
	if err != nil {
		return err
	}
	if cast.ToInt(strings.TrimSpace(fileNumber)) > 0 {
		return nil
	}
	err = remoteBash.SftpDirectory(b.c.Infrastructure.Resource, remoteResroucePath)
	if err != nil {
		return err
	}
	return nil
}

type SystemInfo struct {
	Id      string `json:"id"`
	Os      string `json:"os"`
	Arch    string `json:"arch"`
	Mem     string `json:"mem"`
	Cpu     string `json:"cpu"`
	Gpu     string `json:"gpu"`
	GpuInfo string `json:"gpu_info"`
	Disk    string `json:"disk"`
	Ip      string `json:"ip"`
}

func (b *Baremetal) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Nodes == nil {
		cluster.Nodes = make([]*biz.Node, 0)
	}
	if cluster.NodeGroups == nil {
		cluster.NodeGroups = make([]*biz.NodeGroup, 0)
	}

	// get all node ip
	eg := new(errgroup.Group)
	eg.SetLimit(10)
	mu := new(sync.Mutex)
	nodeIps := utils.RangeIps(cluster.NodeStartIp, cluster.NodeEndIp)
	systemInfos := make([]SystemInfo, 0)
	for _, ip := range nodeIps {
		if cluster.GetNodeByIp(ip) != nil {
			continue
		}
		ip := ip
		eg.Go(func() error {
			systemInfoOutput, err := utils.NewRemoteBash(utils.Server{
				Name:       ip,
				Host:       ip,
				User:       cluster.Username,
				Port:       defaultSHHPort,
				PrivateKey: cluster.PrivateKey,
			}, b.c.Infrastructure.Shell, b.log).ExecShell(SystemInfoShell)
			if err != nil {
				b.log.Errorf("node %s connection refused", ip)
				return nil
			}
			systemInfo := SystemInfo{Ip: ip}
			if err := json.Unmarshal([]byte(systemInfoOutput), &systemInfo); err != nil {
				b.log.Errorf("node %s connection refused", ip)
				return nil
			}
			mu.Lock()
			systemInfos = append(systemInfos, systemInfo)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	// group by os, arch, mem, cpu, gpu, gpu_info
	groupMap := make(map[string]*biz.NodeGroup)
	for _, info := range systemInfos {
		groupKey := fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			info.Os, info.Arch, info.Mem, info.Cpu, info.Gpu, info.GpuInfo)
		if _, exists := groupMap[groupKey]; !exists {
			groupMap[groupKey] = &biz.NodeGroup{
				Id:     uuid.NewString(),
				Type:   biz.NodeGroupType_NORMAL,
				Name:   fmt.Sprintf("group-%s-%s-%s", info.Arch, info.Cpu, info.Mem),
				Os:     info.Os,
				Arch:   getNodeArchByBareMetal(info.Arch),
				Memory: cast.ToInt32(info.Mem),
				Cpu:    cast.ToInt32(info.Cpu),
				Gpu:    cast.ToInt32(info.Gpu),
			}
			if cast.ToInt32(info.Gpu) > 0 {
				groupMap[groupKey].Type = biz.NodeGroupType_GPU_ACCELERATERD
				groupMap[groupKey].GpuSpec = getGPUSpecByBareMetal(strings.ToLower(info.GpuInfo))
			}
		}
		cluster.Nodes = append(cluster.Nodes, &biz.Node{
			Name:           info.Id,
			Ip:             info.Ip,
			User:           cluster.Username,
			SystemDiskSize: cast.ToInt32(info.Disk),
			NodeGroupId:    groupMap[groupKey].Id,
			ClusterId:      cluster.Id,
		})
	}

	// set node group target size
	for _, group := range groupMap {
		nodeNumber := int32(0)
		for _, node := range cluster.Nodes {
			if node.NodeGroupId == group.Id {
				nodeNumber++
			}
		}
		group.TargetSize = nodeNumber
		group.MinSize = nodeNumber
		group.MaxSize = nodeNumber
		group.ClusterId = cluster.Id
		cluster.NodeGroups = append(cluster.NodeGroups, group)
	}
	return nil
}

func (b *Baremetal) ApplyCloudCopilot(ctx context.Context, cluster *biz.Cluster) error {
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
	arch = getNodeArchByBareMetal(strings.TrimSpace(strings.ToLower(arch))).String()
	kubeCtlPath := filepath.Join(userHomePath, b.c.Infrastructure.Resource, arch, "kubernetes", getKubernetesVersion(), "kubectl")
	err = remoteBash.RunWithLogging("install -m 755", kubeCtlPath, "/usr/local/bin/kubectl")
	if err != nil {
		return err
	}
	installfile, err := utils.TransferredMeaning(cluster, filepath.Join(b.c.Infrastructure.Component, Install))
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
	cluster.Config, err = utils.TransferredMeaningString(cluster,
		filepath.Join(b.c.Infrastructure.Component, ClusterConfiguration))
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

func (b *Baremetal) PreInstall(cluster *biz.Cluster) error {
	if cluster.Status != biz.ClusterStatus_STARTING {
		return nil
	}
	for _, node := range cluster.Nodes {
		if node.Role == biz.NodeRole_MASTER {
			clusterJsonByte, err := json.Marshal(cluster)
			if err != nil {
				return err
			}
			err = b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(InstallShell,
				fmt.Sprintf(`'%s'`, string(clusterJsonByte)))
			if err != nil {
				return err
			}
			break
		}
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
		Port:       defaultSHHPort,
		PrivateKey: cluster.PrivateKey,
	}, b.c.Infrastructure.Shell, b.log)
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
