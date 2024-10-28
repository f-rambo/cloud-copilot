package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
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
	"golang.org/x/sync/errgroup"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var ARCH_MAP = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

type ClusterInfrastructure struct {
	log                            *log.Helper
	conf                           *conf.Bootstrap
	oceanPath                      string
	shipPath                       string
	oceanDataTargzPackagePath      string
	installScriptAndStartOceanName string
	startShipName                  string
	downloadAndCopyScriptName      string
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &ClusterInfrastructure{
		log:                            log.NewHelper(logger),
		conf:                           c,
		oceanPath:                      "/app/ocean",
		shipPath:                       "/app/ship",
		oceanDataTargzPackagePath:      "/tmp/oceandata.tar.gz",
		installScriptAndStartOceanName: "install_and_start_ocean.sh",
		startShipName:                  "start_ship.sh",
		downloadAndCopyScriptName:      "download_and_copy.sh",
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
	// get current .ocean package path
	currentOceanFilePath, err := utils.GetPackageStorePathByNames()
	if err != nil {
		return err
	}
	scriptMaps := map[string]string{
		fmt.Sprintf("%s/%s", currentOceanFilePath, cc.installScriptAndStartOceanName): getInstallScriptAndStartOcean(cc.oceanPath, cc.shipPath, conf.EnvBostionHost.String()),
		fmt.Sprintf("%s/%s", currentOceanFilePath, cc.startShipName):                  getShipStartScript(cc.shipPath),
		fmt.Sprintf("%s/%s", currentOceanFilePath, cc.downloadAndCopyScriptName):      getdownloadAndCopyScript(),
	}
	for scriptPath, script := range scriptMaps {
		err = os.WriteFile(scriptPath, []byte(script), 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write %s", scriptPath)
		}
	}

	err = cc.runCommandWithLogging("tar", "-czvf", cc.oceanDataTargzPackagePath, "-C", filepath.Dir(currentOceanFilePath), utils.PackageStoreDirName)
	if err != nil {
		return err
	}

	// scp package to bostion host
	// rm  cc.oceanDataTargzPackagePath
	_, err = remoteBash.Run("rm", "-rf", cc.oceanDataTargzPackagePath)
	if err != nil {
		return err
	}
	err = cc.runCommandWithLogging("scp", "-o", "StrictHostKeyChecking=no", "-r", "-P",
		fmt.Sprintf("%d", cluster.BostionHost.SshPort), cc.oceanDataTargzPackagePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.User, cluster.BostionHost.ExternalIP, cc.oceanDataTargzPackagePath))
	if err != nil {
		return err
	}

	// start ocean by root user
	rootHomePathOutput, err := remoteBash.Run("grep '^root:' /etc/passwd | cut -d: -f6")
	if err != nil {
		return err
	}
	rootUserHomePath := strings.TrimSpace(rootHomePathOutput)
	if rootUserHomePath == "" {
		return errors.New("root user home path is empty")
	}

	err = remoteBash.RunWithLogging("tar", "-xzvf", cc.oceanDataTargzPackagePath, "-C", rootUserHomePath)
	if err != nil {
		return err
	}

	// install ocean and ship to bostion host
	err = remoteBash.RunWithLogging("bash", fmt.Sprintf("%s/%s/%s", rootUserHomePath, utils.PackageStoreDirName, cc.installScriptAndStartOceanName),
		cluster.BostionHost.ARCH, cc.conf.Server.Version, cc.conf.Ship.Version)
	if err != nil {
		return err
	}

	// grpc check bostion host data and resources
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
			_, err = client.SetingIpv4Forward(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			_, err = client.CloseSwap(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			_, err = client.CloseFirewall(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			nodeGroup := cluster.GetNodeGroup(node.NodeGroupID)
			if nodeGroup == nil {
				return errors.New("node group is nil")
			}
			_, err = client.InstallKubeadmKubeletCriO(ctx, &cloudv1alpha1.Cloud{
				Arch:        nodeGroup.ARCH,
				CrioVersion: node.ContainerRuntime,
			})
			if err != nil {
				return err
			}
			_, err = client.AddKubeletServiceAndSettingKubeadmConfig(ctx, &cloudv1alpha1.Cloud{
				KubeadmConfig:  utils.KubeadmConfig,
				KubeletService: utils.KubeletService,
			})
			if err != nil {
				return err
			}
			_, err = client.KubeadmInit(ctx, &cloudv1alpha1.Cloud{
				KubeadmInitConfig:        utils.KubeadmInitConfig,
				KubeadmInitClusterConfig: utils.KubeadmClusterConfig,
			})
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
			_, err = client.KubeadmJoin(ctx, &cloudv1alpha1.Cloud{
				JoinConfig: utils.KubeadmJoinConfig,
			})
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
			_, err = client.KubeadmReset(ctx, &cloudv1alpha1.Cloud{
				KubeadmResetConfig: utils.KubeadmResetConfig,
			})
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
			// check node information
			if node.InternalIP == "" || node.User == "" {
				return errors.New("node required parameter is empty; (InternalIP and User)")
			}
			// check ship server
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

			oceanHomePath, err := utils.GetPackageStorePathByNames()
			if err != nil {
				return err
			}

			// get node arch
			remoteBash := NewBash(Server{
				Name:       node.Name,
				Host:       node.InternalIP,
				User:       node.User,
				Port:       22,
				PrivateKey: cluster.PrivateKey,
			}, cc.log)
			stdout, err := remoteBash.Run("uname -m")
			if err != nil {
				return err
			}
			arch := strings.TrimSpace(stdout)
			if _, ok := ARCH_MAP[arch]; !ok {
				return errors.New("node arch is not supported")
			}

			// kuberentes software
			downloadAndCopyScrip := fmt.Sprintf("%s/%s", cc.shipPath, cc.downloadAndCopyScriptName)
			cloudSowftwareVersion := utils.GetCloudSowftwareVersion(cluster.Version)
			if cloudSowftwareVersion.KubernetesVersion == "" {
				return errors.New("kubernetes version is not supported")
			}
			crioFileName := fmt.Sprintf("crio.%s.v%s.tar.gz", arch, cloudSowftwareVersion.GetCrioLatestVersion())
			output, err := cc.execCommand(
				"bash", downloadAndCopyScrip,
				fmt.Sprintf("https://storage.googleapis.com/cri-o/artifacts/%s", crioFileName),
				fmt.Sprintf("%s/%s", oceanHomePath, crioFileName),
				node.InternalIP, node.User, fmt.Sprintf("/tmp/%s", crioFileName), "22")
			if err != nil {
				return errors.Wrap(err, output)
			}
			kubeadmFileName := "kubeadm"
			output, err = cc.execCommand("bash", downloadAndCopyScrip,
				fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/%s/%s", cloudSowftwareVersion.GetKubeadmLatestVersion(), arch, kubeadmFileName),
				fmt.Sprintf("%s/%s", oceanHomePath, kubeadmFileName),
				node.InternalIP, node.User, fmt.Sprintf("/tmp/%s", kubeadmFileName), "22")
			if err != nil {
				return errors.Wrap(err, output)
			}
			kubeletFileName := "kubelet"
			output, err = cc.execCommand("bash", downloadAndCopyScrip,
				fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/%s/%s", cloudSowftwareVersion.GetKubeletLatestVersion(), arch, kubeletFileName),
				fmt.Sprintf("%s/%s", oceanHomePath, kubeletFileName),
				node.InternalIP, node.User, fmt.Sprintf("/tmp/%s", kubeletFileName), "22")
			if err != nil {
				return errors.Wrap(err, output)
			}

			// ship
			output, err = remoteBash.Run("mkdir -p", cc.shipPath)
			if err != nil {
				return errors.Wrap(err, output)
			}
			shipArchPath := fmt.Sprintf("%s/%s", cc.shipPath, ARCH_MAP[arch])
			if !utils.IsFileExist(shipArchPath) {
				return errors.New("ship arch is not exist")
			}
			tmpShipPath := "/tmp/ship"
			_, err = remoteBash.Run("rm", "-rf", tmpShipPath)
			if err != nil {
				return err
			}
			output, err = cc.execCommand("scp", "-o", "StrictHostKeyChecking=no", "-r", shipArchPath, fmt.Sprintf("%s@%s:%s", node.User, node.InternalIP, tmpShipPath))
			if err != nil {
				return errors.Wrap(err, output)
			}
			// cp ship to app dir
			output, err = remoteBash.Run(fmt.Sprintf("cp -r %s/* %s", tmpShipPath, cc.shipPath))
			if err != nil {
				return errors.Wrap(err, output)
			}
			// scp ship start script to node in the bostion host
			shipStartScriptPath := fmt.Sprintf("%s/%s", oceanHomePath, cc.startShipName)
			if !utils.IsFileExist(shipStartScriptPath) {
				return errors.New("ship start script is not exist")
			}
			nodeShipStartScriptTmpPath := fmt.Sprintf("/tmp/%s", cc.startShipName)
			// rm nodeShipStartScriptTmpPath
			_, err = remoteBash.Run("rm", "-rf", nodeShipStartScriptTmpPath)
			if err != nil {
				return err
			}
			output, err = cc.execCommand("scp", "-o", "StrictHostKeyChecking=no", "-r", shipStartScriptPath,
				fmt.Sprintf("%s@%s:%s", node.User, node.InternalIP, nodeShipStartScriptTmpPath))
			if err != nil {
				return errors.Wrap(err, output)
			}
			output, err = remoteBash.Run("cp -r", nodeShipStartScriptTmpPath, fmt.Sprintf("%s/%s", cc.shipPath, cc.startShipName))
			if err != nil {
				return errors.Wrap(err, output)
			}
			// run ship start script
			err = remoteBash.RunWithLogging("bash", fmt.Sprintf("%s/%s", cc.shipPath, cc.startShipName))
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

// exec command
func (c *ClusterInfrastructure) execCommand(command string, args ...string) (output string, err error) {
	c.log.Info("exec command: ", fmt.Sprintf("%s %s", command, strings.Join(args, " ")))

	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		return "", errors.Wrapf(err, "command failed: %s\nstdout: %s\nstderr: %s", command, stdoutStr, stderrStr)
	}

	if stderrStr != "" {
		c.log.Warnf("command wrote to stderr: %s", stderrStr)
	}

	return stdoutStr, nil
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
