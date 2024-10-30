package infrastructure

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	cloudv1alpha1 "github.com/f-rambo/ocean/api/cloud/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/api/common"
	systemv1alpha1 "github.com/f-rambo/ocean/api/system/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var ARCH_MAP = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

var (
	ServiceShell string = "service.sh"
	SyncShell    string = "sync.sh"
	RemoteShell  string = "remote.sh"
)

type ClusterInfrastructure struct {
	log  *log.Helper
	conf *conf.Bootstrap
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &ClusterInfrastructure{
		log:  log.NewHelper(logger),
		conf: c,
	}
}

func (c *ClusterInfrastructure) GetRegions(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Type.IsCloud() {
		return nil
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 || cluster.Type == biz.ClusterTypeAWSEks {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		return awsCloud.GetAvailabilityZones(ctx)
	}
	if cluster.Type == biz.ClusterTypeAliCloudEcs || cluster.Type == biz.ClusterTypeAliCloudAks {
		alicloud, err := NewAlicloud(cluster, c.log)
		if err != nil {
			return err
		}
		return alicloud.GetAvailabilityZones()
	}
	return errors.New("cluster type is not supported")
}

func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) (err error) {
	if !cluster.Type.IsCloud() {
		return nil
	}
	if len(cluster.GetCloudResource(biz.ResourceTypeAvailabilityZones)) == 0 {
		return errors.New("availability zones is empty")
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		err = awsCloud.CreateNetwork(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.SetByNodeGroups(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ImportKeyPair(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ManageInstance(ctx)
		if err != nil {
			return err
		}
		return awsCloud.ManageBostionHost(ctx)
	}
	return errors.New("cluster type is not supported")
}

func (c *ClusterInfrastructure) Stop(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.Type.IsCloud() {
		return nil
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 {
		awsCloud, err := NewAwsCloud(ctx, cluster, c.conf, c.log)
		if err != nil {
			return err
		}
		err = awsCloud.ManageInstance(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.ManageBostionHost(ctx)
		if err != nil {
			return err
		}
		err = awsCloud.DeleteKeyPair(ctx)
		if err != nil {
			return err
		}
		return awsCloud.DeleteNetwork(ctx)
	}
	return errors.New("cluster type is not supported")
}

func (cc *ClusterInfrastructure) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.BostionHost.User == "" {
		return errors.New("bostion host username is empty")
	}
	if cluster.BostionHost.ExternalIP == "" {
		return errors.New("bostion host external ip is empty")
	}
	if cluster.BostionHost.SshPort == 0 {
		cluster.BostionHost.SshPort = 22
	}
	remoteBash := NewBash(Server{
		Name:       "bostion host",
		Host:       cluster.BostionHost.ExternalIP,
		User:       cluster.BostionHost.User,
		Port:       cluster.BostionHost.SshPort,
		PrivateKey: cluster.PrivateKey,
	}, cc.log)
	stdout, err := remoteBash.Run("uname -m")
	if err != nil {
		return err
	}
	arch := strings.TrimSpace(stdout)
	if _, ok := ARCH_MAP[arch]; !ok {
		return errors.New("bostion host arch is not supported")
	}
	cluster.BostionHost.ARCH = ARCH_MAP[arch]
	if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, SyncShell)) {
		return errors.New("sync shell script is not exist")
	}
	oceanHomePath, err := utils.GetPackageStorePathByNames()
	if err != nil {
		return err
	}
	err = cc.runCommandWithLogging("bash", utils.MergePath(cc.conf.Server.Shell, SyncShell),
		cluster.BostionHost.ExternalIP,
		cast.ToString(cluster.BostionHost.SshPort),
		cluster.BostionHost.User,
		cluster.PrivateKey,
		oceanHomePath,
		utils.MergePath(filepath.Dir(oceanHomePath), utils.ShipPackageStoreDirName),
		cc.conf.Server.Resource,
		cc.conf.Server.Shell,
	)
	if err != nil {
		return err
	}
	if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, RemoteShell)) {
		return errors.New("remote shell script is not exist")
	}
	if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, ServiceShell)) {
		return errors.New("service shell script is not exist")
	}
	err = cc.runCommandWithLogging("bash", utils.MergePath(cc.conf.Server.Shell, RemoteShell),
		cluster.BostionHost.User,
		cluster.BostionHost.ExternalIP,
		cast.ToString(cluster.BostionHost.SshPort),
		cluster.PrivateKey,
		utils.MergePath(cc.conf.Server.Shell, ServiceShell),
		conf.EnvBostionHost.String(),
		cc.conf.Server.Version,
		cc.conf.Ship.Version,
		cc.conf.Server.Resource,
		cc.conf.Server.Shell,
	)
	if err != nil {
		return err
	}
	conn, err := grpc.DialInsecure(ctx,
		grpc.WithEndpoint(fmt.Sprintf("%s:%d", cluster.BostionHost.ExternalIP, utils.GetPortByAddr(cc.conf.Server.GRPC.Addr))),
		grpc.WithMiddleware(mmd.Client()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := clusterv1alpha1.NewClusterInterfaceClient(conn)
	appInfo := utils.GetFromContext(ctx)
	for k, v := range appInfo {
		ctx = metadata.AppendToClientContext(ctx, k, v)
	}
	msg, err := client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if msg.Reason != common.ErrorReason_SUCCEED {
		return errors.New(msg.Message)
	}
	return nil
}

func (cc *ClusterInfrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
			// grpc to ship server
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, cc.conf.Ship.GrpcPort)),
				grpc.WithMiddleware(mmd.Client()),
			)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := cloudv1alpha1.NewCloudInterfaceClient(conn)
			appInfo := utils.GetFromContext(ctx)
			for k, v := range appInfo {
				ctx = metadata.AppendToClientContext(ctx, k, v)
			}
			_, err = client.NodeInit(ctx, &cloudv1alpha1.Cloud{
				NodeId:   node.ID,
				NodeName: node.Name,
			})
			if err != nil {
				return err
			}
			nodeGroup := cluster.GetNodeGroup(node.NodeGroupID)
			if nodeGroup == nil {
				return errors.New("node group is nil")
			}
			_, err = client.InstallKubeadmKubeletCriO(ctx, &cloudv1alpha1.Cloud{
				ClusterVersion: cluster.Version,
			})
			if err != nil {
				return err
			}
			_, err = client.KubeadmInit(ctx, &cloudv1alpha1.Cloud{})
			if err != nil {
				return err
			}
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	return cc.RemoveNodes(ctx, cluster, cluster.Nodes)
}

func (cc *ClusterInfrastructure) AddNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		node := node
		errGroup.Go(func() error {
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, cc.conf.Ship.GrpcPort)),
				grpc.WithMiddleware(
					mmd.Client(),
				),
			)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := cloudv1alpha1.NewCloudInterfaceClient(conn)
			appInfo := utils.GetFromContext(ctx)
			for k, v := range appInfo {
				ctx = metadata.AppendToClientContext(ctx, k, v)
			}
			_, err = client.KubeadmJoin(ctx, &cloudv1alpha1.Cloud{})
			if err != nil {
				return err
			}
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) RemoveNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		node := node
		errGroup.Go(func() error {
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, cc.conf.Ship.GrpcPort)),
				grpc.WithMiddleware(
					mmd.Client(),
				),
			)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := cloudv1alpha1.NewCloudInterfaceClient(conn)
			appInfo := utils.GetFromContext(ctx)
			for k, v := range appInfo {
				ctx = metadata.AppendToClientContext(ctx, k, v)
			}
			_, err = client.KubeadmReset(ctx, &cloudv1alpha1.Cloud{})
			if err != nil {
				return err
			}
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Distribute the “ship server” to each node in the bostion host
func (cc *ClusterInfrastructure) DistributeDaemonApp(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, _ := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
			if node.InternalIP == "" || node.User == "" {
				return errors.New("node required parameter is empty; (InternalIP and User)")
			}
			conn, err := grpc.DialInsecure(ctx, grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, cc.conf.Ship.GrpcPort)))
			if err != nil {
				return err
			}
			defer conn.Close()
			client := systemv1alpha1.NewSystemInterfaceClient(conn)
			msg, err := client.Ping(ctx, &emptypb.Empty{})
			if err == nil && msg.Reason == common.ErrorReason_SUCCEED {
				return nil
			}
			if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, SyncShell)) {
				return errors.New("sync shell script is not exist")
			}
			oceanHomePath, err := utils.GetPackageStorePathByNames()
			if err != nil {
				return err
			}
			err = cc.runCommandWithLogging("bash", utils.MergePath(cc.conf.Server.Shell, SyncShell),
				node.InternalIP,
				"22",
				node.User,
				cluster.PrivateKey,
				oceanHomePath,
				utils.MergePath(filepath.Dir(oceanHomePath), utils.ShipPackageStoreDirName),
				cc.conf.Server.Resource,
				cc.conf.Server.Shell,
			)
			if err != nil {
				return err
			}
			if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, RemoteShell)) {
				return errors.New("remote shell script is not exist")
			}
			if !utils.IsFileExist(utils.MergePath(cc.conf.Server.Shell, ServiceShell)) {
				return errors.New("service shell script is not exist")
			}
			err = cc.runCommandWithLogging("bash", utils.MergePath(cc.conf.Server.Shell, RemoteShell),
				node.User,
				node.InternalIP,
				"22",
				cluster.PrivateKey,
				utils.MergePath(cc.conf.Server.Shell, ServiceShell),
				conf.EnvCluster.String(),
				cc.conf.Server.Version,
				cc.conf.Ship.Version,
				cc.conf.Server.Resource,
				cc.conf.Server.Shell,
			)
			if err != nil {
				return err
			}
			msg, err = client.Ping(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			if msg.Reason != common.ErrorReason_SUCCEED {
				return errors.New(msg.Message)
			}
			cc.log.Infof("ship server is installed on node %s", node.Name)
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	// cloud local
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		nodegroup := cluster.NewNodeGroup()
		node := node
		errGroup.Go(func() error {
			conn, err := grpc.DialInsecure(ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, cc.conf.Ship.GrpcPort)),
				grpc.WithMiddleware(mmd.Client()),
			)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := systemv1alpha1.NewSystemInterfaceClient(conn)
			appInfo := utils.GetFromContext(ctx)
			for k, v := range appInfo {
				ctx = metadata.AppendToClientContext(ctx, k, v)
			}
			systemInfo, err := client.GetSystem(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			// node group
			nodegroup.ARCH = systemInfo.Arch
			nodegroup.CPU = systemInfo.Cpu
			nodegroup.Memory = systemInfo.Memory
			nodegroup.GPU = systemInfo.Gpu
			nodegroup.OS = systemInfo.Os
			nodegroup.GpuSpec = systemInfo.GpuSpec
			nodegroup.DataDisk = systemInfo.DataDisk
			cluster.GenerateNodeGroupName(nodegroup)
			exitsNodeGroup := cluster.GetNodeGroupByName(nodegroup.Name)
			if exitsNodeGroup == nil {
				cluster.NodeGroups = append(cluster.NodeGroups, nodegroup)
			} else {
				nodegroup.ID = exitsNodeGroup.ID
			}
			// node
			node.Kernel = systemInfo.Kernel
			node.ContainerRuntime = systemInfo.Container
			node.Kubelet = systemInfo.Kubelet
			node.KubeProxy = systemInfo.KubeProxy
			node.InternalIP = systemInfo.InternalIp
			node.NodeGroupID = nodegroup.ID
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

// log
func (c *ClusterInfrastructure) Write(content []byte) (n int, err error) {
	c.log.Info(string(content))
	return len(content), nil
}

func (c *ClusterInfrastructure) runCommandWithLogging(command string, args ...string) error {
	c.log.Info("exec command: ", fmt.Sprintf("%s %s", command, strings.Join(args, " ")))
	cmd := exec.Command(command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	// use scanner to read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			c.log.Info(scanner.Text())
		}
	}()

	// use scanner to read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			c.log.Warn(scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "command failed")
	}

	return nil
}
