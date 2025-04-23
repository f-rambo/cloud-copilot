package infrastructure

import (
	"context"
	"encoding/json"
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
		User:       node.Username,
		Port:       defaultSHHPort,
		PrivateKey: cluster.PrivateKey,
	}, b.c.Infrastructure.Shell, b.log)
}

func (b *Baremetal) initNode(cluster *biz.Cluster, node *biz.Node) error {
	remoteBash := b.getClusterNodeRemoteBash(cluster, node)
	err := remoteBash.ExecShellLogging(NodeInitShell, node.Name)
	if err != nil {
		return err
	}
	userHomePath, err := remoteBash.GetUserHome()
	if err != nil {
		return err
	}
	err = remoteBash.ExecShellLogging(
		ComponentShell,
		filepath.Join(userHomePath, b.c.Infrastructure.Resource),
		cluster.ImageRepository,
		getKubernetesVersion(b.c.Infrastructure.Resource),
		getContainerdVersion(b.c.Infrastructure.Resource),
		getRuncVersion(b.c.Infrastructure.Resource),
	)
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
	fileNumber, err := remoteBash.Run(fmt.Sprintf("test -d %s && echo 1 || echo 0", remoteResroucePath))
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
	Id                 string              `json:"id"`
	Os                 string              `json:"os"`
	Arch               string              `json:"arch"`
	Mem                string              `json:"mem"`
	Cpu                string              `json:"cpu"`
	Gpu                string              `json:"gpu"`
	GpuInfo            string              `json:"gpu_info"`
	Ip                 string              `json:"ip"`
	UnpartitionedDisks []UnpartitionedDisk `json:"unpartitioned_disks"`
}

type UnpartitionedDisk struct {
	Name   string `json:"name"`
	Device string `json:"device"`
	Size   string `json:"size"`
}

func (b *Baremetal) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	// get all node ip
	eg := new(errgroup.Group)
	eg.SetLimit(10)
	mu := new(sync.Mutex)
	systemInfos := make([]SystemInfo, 0)
	for _, node := range cluster.Nodes {
		if node.Status != biz.NodeStatus_NODE_FINDING {
			continue
		}
		ip := node.Ip
		eg.Go(func() error {
			systemInfoOutput, err := utils.NewRemoteBash(utils.Server{
				Name:       ip,
				Host:       ip,
				User:       cluster.NodeUsername,
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
	for _, info := range systemInfos {
		nodeGroupId := uuid.NewString()
		nodeGroup := &biz.NodeGroup{
			Id:     nodeGroupId,
			Type:   biz.NodeGroupType_NORMAL,
			Name:   fmt.Sprintf("group-%s-%s-%s", info.Arch, info.Cpu, info.Mem),
			Os:     info.Os,
			Arch:   getNodeArchByBareMetal(info.Arch),
			Memory: cast.ToInt32(info.Mem),
			Cpu:    cast.ToInt32(info.Cpu),
			Gpu:    cast.ToInt32(info.Gpu),
		}
		if cast.ToInt32(info.Gpu) > 0 {
			nodeGroup.Type = biz.NodeGroupType_GPU_ACCELERATERD
			nodeGroup.GpuSpec = getGPUSpecByBareMetal(strings.ToLower(info.GpuInfo))
		}
		clusterNg := cluster.GetNodeGroupByUniqueKey(nodeGroup.UniqueKey())
		if clusterNg == nil {
			cluster.AddNodeGroup(nodeGroup)
		} else {
			nodeGroup.Id = clusterNg.Id
		}
		clusterNode := cluster.GetNodeByIp(info.Ip)
		for _, disk := range info.UnpartitionedDisks {
			clusterNode.AddDisk(&biz.Disk{
				Name:   disk.Name,
				Device: disk.Device,
				Size:   cast.ToInt32(disk.Size),
			})
		}
		clusterNode.NodeGroupId = nodeGroup.Id
	}
	for _, node := range cluster.Nodes {
		if node.NodeGroupId == "" {
			cluster.DeleteNode(node)
		}
	}
	return nil
}

func (b *Baremetal) Install(ctx context.Context, cluster *biz.Cluster) error {
	masterNode := cluster.GetSingleMasterNode()
	err := b.migrateResources(cluster, masterNode)
	if err != nil {
		return err
	}
	err = b.initNode(cluster, masterNode)
	if err != nil {
		return err
	}
	err = b.getClusterNodeRemoteBash(cluster, masterNode).ExecShellLogging(
		ClusterInitShell,
		getKubernetesVersion(b.c.Infrastructure.Resource),
	)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.Ip == masterNode.Ip {
			continue
		}
		err = b.joinCluster(cluster, node)
		if err != nil {
			return err
		}
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
			err = b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(CloudCopilotInstallShell,
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
		if node.Status == biz.NodeStatus_NODE_PENDING {
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
	err = b.initNode(cluster, node)
	if err != nil {
		return err
	}
	var token, caHash string
	masterNode := cluster.GetSingleMasterNode()
	masterNodeRemoteBash := b.getClusterNodeRemoteBash(cluster, masterNode)
	caHash, err = masterNodeRemoteBash.ExecShell(CluasterCaTokenShell, GetCaHash)
	if err != nil {
		return err
	}
	token, err = masterNodeRemoteBash.ExecShell(CluasterCaTokenShell, GetToken)
	if err != nil {
		return err
	}
	if node.Role == biz.NodeRole_MASTER {
		return b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(ClusterJoinShell, caHash, token)
	}
	return b.getClusterNodeRemoteBash(cluster, node).ExecShellLogging(ClusterJoinShell, caHash, token, ClusterController)
}

func (b *Baremetal) uninstallNode(cluster *biz.Cluster, node *biz.Node) error {
	return utils.NewRemoteBash(utils.Server{
		Name:       node.Name,
		Host:       node.Ip,
		User:       node.Username,
		Port:       defaultSHHPort,
		PrivateKey: cluster.PrivateKey,
	}, b.c.Infrastructure.Shell, b.log).ExecShellLogging(ClusterResetShell)
}
