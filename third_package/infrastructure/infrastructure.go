package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cloudv1alpha1 "github.com/f-rambo/ocean/api/cloud/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
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

type ClusterInfrastructure struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &ClusterInfrastructure{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (c *ClusterInfrastructure) GetRegions(ctx context.Context, cluster *biz.Cluster) ([]string, error) {
	switch cluster.GetType() {
	case biz.ClusterTypeAliCloudEcs:
		return NewAlicloud(cluster).GetRegions()
	case biz.ClusterTypeAliCloudAks:
		return NewAlicloud(cluster).GetRegions()
	case biz.ClusterTypeAWSEc2:
		return NewAwsCloud(cluster).GetRegions()
	case biz.ClusterTypeAWSEks:
		return NewAwsCloud(cluster).GetRegions()
	}
	return nil, errors.New("cluster type is not supported")
}

func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) (err error) {
	if !cluster.GetType().IsCloud() {
		return nil
	}
	var startFunc PulumiFunc
	output := ""
	switch cluster.GetType() {
	case biz.ClusterTypeAliCloudEcs:
		startFunc = NewAlicloud(cluster).Start
	case biz.ClusterTypeAliCloudAks:
		startFunc = NewAlicloud(cluster).StartAks
	case biz.ClusterTypeAWSEc2:
		startFunc = NewAwsCloud(cluster).Start
	case biz.ClusterTypeAWSEks:
		startFunc = NewAwsCloud(cluster).StartEks
	}
	if startFunc == nil {
		return errors.New("not supported cluster type")
	}
	output, err = c.pulumiExec(ctx, cluster, startFunc)
	if err != nil {
		return err
	}
	err = c.parseOutput(cluster, output)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterInfrastructure) Stop(ctx context.Context, cluster *biz.Cluster) error {
	if !cluster.GetType().IsCloud() {
		return nil
	}
	_, err := c.pulumiExec(ctx, cluster, CleanFunc)
	if err != nil {
		return err
	}
	return nil
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
	remoteBash, err := NewRemoteBash(ctx, Server{
		Name:       "bostion host",
		Host:       cluster.BostionHost.ExternalIP,
		User:       cluster.BostionHost.User,
		Port:       cluster.BostionHost.SshPort,
		PrivateKey: cluster.PrivateKey,
	}, cc.log)
	if err != nil {
		return err
	}
	defer remoteBash.Close()
	if cluster.BostionHost.ARCH == "" {
		stdout, err := remoteBash.Run("uname -m")
		if err != nil {
			return err
		}
		arch := strings.TrimSpace(stdout)
		if _, ok := ARCH_MAP[arch]; !ok {
			return errors.New("bostion host arch is not supported")
		}
		cluster.BostionHost.ARCH = ARCH_MAP[arch]
	}
	// get current .ocean package path
	currentOceanFilePath, err := utils.GetPackageStorePathByNames()
	if err != nil {
		return err
	}
	// packing and storing files, this is ocean data and resources
	err = cc.runCommandWithLogging("tar", "-czvf", oceanDataTargzPackagePath, "-C", filepath.Dir(currentOceanFilePath), utils.PackageStoreDirName)
	if err != nil {
		return err
	}
	// .ocean package generating sha256sum
	stdout, err := cc.execCommand("sha256sum", oceanDataTargzPackagePath)
	if err != nil {
		return err
	}
	err = os.WriteFile(oceanDataTsha256sumFilePath, []byte(stdout), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write sha256sum file")
	}
	// scp .ocean package to bostion host
	err = cc.runCommandWithLogging("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.SshPort), oceanDataTargzPackagePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.User, cluster.BostionHost.ExternalIP, oceanDataTargzPackagePath))
	if err != nil {
		return err
	}
	// scp .ocean package sha256sum to bostion host
	err = cc.runCommandWithLogging("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.SshPort), oceanDataTsha256sumFilePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.User, cluster.BostionHost.ExternalIP, oceanDataTsha256sumFilePath))
	if err != nil {
		return err
	}
	// install ocean and ship to bostion host
	err = remoteBash.RunWithLogging(installScript, cluster.BostionHost.ARCH, cc.c.Server.Version, cc.c.Server.ShipVersion)
	if err != nil {
		return err
	}

	// grpc check bostion host data and resources
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(fmt.Sprintf("%s:%d", cluster.BostionHost.ExternalIP, 9000)),
		grpc.WithMiddleware(
			mmd.Client(),
		),
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
	_, err = client.CheckBostionHost(ctx, &clusterv1alpha1.CheckBostionHostRequest{
		Arch:                               cluster.BostionHost.ARCH,
		OceanVersion:                       cc.c.Server.Version,
		ShipVersion:                        cc.c.Server.ShipVersion,
		OceanDataTarGzPackagePath:          oceanDataTargzPackagePath,
		OceanDataTarGzPackageSha256SumPath: oceanDataTsha256sumFilePath,
		OceanPath:                          oceanPath,
		ShipPath:                           shipPath,
	})
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
			err := cc.downloadAndCopyK8sSoftware(ctx, cluster, node)
			if err != nil {
				return err
			}
			// grpc to ship server
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, 9000)),
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
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, 9001)),
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
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, 9001)),
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
			if node.InternalIP == "" {
				return errors.New("node internal ip is empty")
			}
			if node.User == "" {
				return errors.New("node user is empty")
			}
			if node.SshPort == 0 {
				node.SshPort = 22
			}
			if node.GrpcPort == 0 {
				node.GrpcPort = 9001
			}

			// check ship server
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, node.GrpcPort)),
			)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := systemv1alpha1.NewSystemInterfaceClient(conn)
			_, err = client.Ping(ctx, &emptypb.Empty{})
			if err == nil {
				// ship server is already installed
				return nil
			}

			// get node arch
			remoteBash, err := NewRemoteBash(ctx, Server{
				Name:       node.Name,
				Host:       node.InternalIP,
				User:       node.User,
				Port:       node.SshPort,
				PrivateKey: cluster.PrivateKey,
			}, cc.log)
			if err != nil {
				return err
			}
			defer remoteBash.Close()
			stdout, err := remoteBash.Run("uname -m")
			if err != nil {
				return err
			}
			arch := strings.TrimSpace(string(stdout))
			if _, ok := ARCH_MAP[arch]; !ok {
				return errors.New("node arch is not supported")
			}

			// get ship arch path
			shipArchPath := fmt.Sprintf("%s/%s", shipPath, ARCH_MAP[arch])
			if !utils.IsFileExist(shipArchPath) {
				return errors.New("ship arch is not exist")
			}
			// scp ship arch to node
			output, err := cc.execCommand("scp", "-r", "-P", fmt.Sprintf("%d", node.SshPort), shipArchPath,
				fmt.Sprintf("%s@%s:%s", node.User, node.InternalIP, shipPath))
			if err != nil {
				return errors.Wrap(err, string(output))
			}
			// run ship start script
			err = remoteBash.RunWithLogging(shipStartScript)
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

func (cc *ClusterInfrastructure) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		nodegroup := cluster.GetNodeGroup(node.NodeGroupID)
		if nodegroup == nil {
			nodegroup = cluster.NewNodeGroup()
		}
		node := node
		errGroup.Go(func() error {
			conn, err := grpc.DialInsecure(
				ctx,
				grpc.WithEndpoint(fmt.Sprintf("%s:%d", node.InternalIP, 9001)),
				grpc.WithMiddleware(
					mmd.Client(),
				),
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
			// node
			node.SystemDisk = systemInfo.DataDisk
			node.GpuSpec = systemInfo.GpuSpec
			node.DataDisk = systemInfo.DataDisk
			node.Kernel = systemInfo.Kernel
			node.ContainerRuntime = systemInfo.Container
			node.Kubelet = systemInfo.Kubelet
			node.KubeProxy = systemInfo.KubeProxy
			node.InternalIP = systemInfo.InternalIp
			node.NodeGroupID = nodegroup.ID
			return nil
		})
		cluster.NodeGroups = append(cluster.NodeGroups, nodegroup)
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	// Node group De-duplication
	nodeGroupMap := make(map[string]*biz.NodeGroup)
	nodeGroupIdMaps := make(map[string][]string)
	for _, nodeGroup := range cluster.NodeGroups {
		key := fmt.Sprintf("%s-%d-%.0f-%d-%s", nodeGroup.ARCH, nodeGroup.CPU, nodeGroup.Memory, nodeGroup.GPU, nodeGroup.OS)
		if _, exists := nodeGroupMap[key]; !exists {
			nodeGroup.Name = key
			nodeGroupMap[key] = nodeGroup
		}
		nodeGroupIdMaps[key] = append(nodeGroupIdMaps[key], nodeGroup.ID)
	}

	// Update cluster.NodeGroups with de-duplicated node groups
	cluster.NodeGroups = make([]*biz.NodeGroup, 0, len(nodeGroupMap))
	for _, nodeGroup := range nodeGroupMap {
		cluster.NodeGroups = append(cluster.NodeGroups, nodeGroup)
	}

	// Update node group id
	for _, node := range cluster.Nodes {
		for _, nodeGroupIDs := range nodeGroupIdMaps {
			for _, id := range nodeGroupIDs {
				if node.NodeGroupID == id {
					node.NodeGroupID = nodeGroupIDs[0]
					break
				}
			}
		}
	}
	return nil
}

// https://github.com/cri-o/cri-o/releases
func (c *ClusterInfrastructure) downloadAndCopyK8sSoftware(_ context.Context, cluster *biz.Cluster, node *biz.Node) error {
	cloudSowftwareVersion := utils.GetCloudSowftwareVersion(cluster.Version)
	crioVersion := cloudSowftwareVersion.GetCrioLatestVersion()
	if crioVersion == "" {
		return errors.New("crio version is empty")
	}
	nodeGroup := cluster.GetNodeGroup(node.NodeGroupID)
	if nodeGroup == nil {
		return errors.New("node group is nil")
	}
	crioFileName := fmt.Sprintf("crio.%s.v%s.tar.gz", nodeGroup.ARCH, crioVersion)
	crioDownloadUrl := fmt.Sprintf("https://storage.googleapis.com/cri-o/artifacts/%s", crioFileName)
	output, err := exec.Command("echo", downloadAndCopyScript, "|",
		"bash -s", crioDownloadUrl, crioFileName, node.InternalIP, node.User, fmt.Sprintf("%d", node.SshPort), fmt.Sprintf("/tmp/%s", crioFileName)).
		CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	kubeadmVersion := cloudSowftwareVersion.GetKubeadmLatestVersion()
	if kubeadmVersion == "" {
		return errors.New("kubeadm version is empty")
	}
	kubeadmFileName := "kubeadm"
	kubeadmDownloadUrl := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/%s/%s", kubeadmVersion, nodeGroup.ARCH, kubeadmFileName)
	output, err = exec.Command("echo", downloadAndCopyScript, "|",
		"bash -s", kubeadmDownloadUrl, kubeadmFileName, node.InternalIP, node.User, fmt.Sprintf("%d", node.SshPort), fmt.Sprintf("/tmp/%s", kubeadmFileName)).
		CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	kubeletVersion := cloudSowftwareVersion.GetKubeletLatestVersion()
	if kubeletVersion == "" {
		return errors.New("kubelet version is empty")
	}
	kubeletFileName := "kubelet"
	kubeletDownloadUrl := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/%s/%s", kubeletVersion, nodeGroup.ARCH, kubeletFileName)
	output, err = exec.Command("echo", downloadAndCopyScript, "|",
		"bash -s", kubeletDownloadUrl, kubeletFileName, node.InternalIP, node.User, fmt.Sprintf("%d", node.SshPort), fmt.Sprintf("/tmp/%s", kubeletFileName)).
		CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	return nil
}

// preview is true, only preview the pulumi resources
func (c *ClusterInfrastructure) pulumiExec(ctx context.Context, cluster *biz.Cluster, pulumiFunc PulumiFunc, preview ...bool) (output string, err error) {
	pulumiObj := NewPulumiAPI(ctx, c)
	if cluster.Type == biz.ClusterTypeAliCloudEcs {
		pulumiObj.ProjectName(AlicloudProjectName).
			StackName(AlicloudStackName).
			Plugin([]PulumiPlugin{
				{Kind: PulumiAlicloud, Version: PulumiAlicloudVersion},
				{Kind: PulumiKubernetes, Version: PulumiKubernetesVersion},
			}...).
			Env(map[string]string{
				"ALICLOUD_ACCESS_KEY": cluster.AccessID,
				"ALICLOUD_SECRET_KEY": cluster.AccessKey,
				"ALICLOUD_REGION":     cluster.Region,
			})
	}
	if cluster.Type == biz.ClusterTypeAWSEc2 {
		pulumiObj.ProjectName(AwsProject).
			StackName(AwsStack).
			Plugin([]PulumiPlugin{
				{Kind: PulumiAws, Version: PulumiAwsVersion},
				{Kind: PulumiKubernetes, Version: PulumiKubernetesVersion},
			}...).
			Env(map[string]string{
				"AWS_ACCESS_KEY_ID":     cluster.AccessID,
				"AWS_SECRET_ACCESS_KEY": cluster.AccessKey,
				"AWS_DEFAULT_REGION":    cluster.Region,
			})
	}
	pulumiObj.RegisterDeployFunc(pulumiFunc)
	if len(preview) > 0 && preview[0] {
		output, err = pulumiObj.Preview(ctx)
	} else {
		output, err = pulumiObj.Up(ctx)
	}
	if err != nil {
		return "", err
	}
	return output, err
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
		return stdoutStr, errors.WithMessage(errors.New(stderrStr), "command failed")
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

	var stderrBuffer bytes.Buffer

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
			line := scanner.Text()
			c.log.Error("command failed: ", line)
			stderrBuffer.WriteString(line + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "command failed")
	}

	// if stderr is not empty, return error
	if stderrBuffer.Len() > 0 {
		return errors.Errorf("command wrote to stderr: %s", stderrBuffer.String())
	}

	return nil
}

type KeyType int

const (
	InstanceID KeyType = iota
	InstanceUser
	InstanceInternalIP
	InstancePublicIP
	BostionHostInstanceID
	ClusterCloudID
	Connections
	CertificateAuthority
	CloudNodeGroupID
	LoadBalancerID
	SecurityGroupIDs
)

func GetKey(keyType KeyType, name ...string) string {
	switch keyType {
	case InstanceID:
		return fmt.Sprintf("node-%s-id", name[0])
	case InstanceUser:
		return fmt.Sprintf("node-%s-user", name[0])
	case InstanceInternalIP:
		return fmt.Sprintf("node-%s-internal-ip", name[0])
	case InstancePublicIP:
		return fmt.Sprintf("node-%s-public-ip", name[0])
	case BostionHostInstanceID:
		return "bostion-host-instance-id"
	case ClusterCloudID:
		return "cluster-cloud-id"
	case Connections:
		return "connections"
	case CertificateAuthority:
		return "certificate-authority"
	case CloudNodeGroupID:
		return fmt.Sprintf("cloud-nodegroup-id-%s", name[0])
	case LoadBalancerID:
		return "load-balancer-id"
	case SecurityGroupIDs:
		return "security-group-ids"
	default:
		return ""
	}
}

// Parse output const
func (c *ClusterInfrastructure) parseOutput(cluster *biz.Cluster, output string) error {
	outputMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(output), &outputMap)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal output")
	}
	data := make(map[string]string)
	for k, v := range outputMap {
		if v == nil {
			continue
		}
		m := make(map[string]interface{})
		vJson, err := json.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "failed to marshal value")
		}
		err = json.Unmarshal(vJson, &m)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal value")
		}
		if _, ok := m["Value"]; !ok {
			continue
		}
		if k == GetKey(SecurityGroupIDs) {
			securityGroupIDs := make([]string, 0)
			securityGroupIDsJson, err := json.Marshal(m["Value"])
			if err != nil {
				return errors.Wrap(err, "failed to marshal security group ids")
			}
			err = json.Unmarshal(securityGroupIDsJson, &securityGroupIDs)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal security group ids")
			}
			cluster.SecurityGroupIDs = strings.Join(securityGroupIDs, ",")
			continue
		}
		data[k] = cast.ToString(m["Value"])
	}
	clusterCloudID, ok := data[GetKey(ClusterCloudID)]
	if ok {
		cluster.CloudID = clusterCloudID
	}
	connections, ok := data[GetKey(Connections)]
	if ok {
		cluster.Connections = connections
	}
	certificateAuthority, ok := data[GetKey(CertificateAuthority)]
	if ok {
		cluster.CertificateAuthority = certificateAuthority
	}
	for _, v := range cluster.NodeGroups {
		cloudNodeGroupID, ok := data[GetKey(CloudNodeGroupID, v.Name)]
		if ok {
			v.CloudNoodGroupID = cloudNodeGroupID
		}
	}
	loadBalancerID, ok := data[GetKey(LoadBalancerID)]
	if ok {
		cluster.LoadBalancerID = loadBalancerID
	}
	for _, node := range cluster.Nodes {
		instanceID, ok := data[GetKey(InstanceID, node.Name)]
		if !ok {
			continue
		}
		node.InstanceID = instanceID
		node.InternalIP = data[GetKey(InstanceInternalIP, node.Name)]
		node.ExternalIP = data[GetKey(InstancePublicIP, node.Name)]
		node.User = data[GetKey(InstanceUser, node.Name)]
	}
	bostionHostInstanceID, ok := data[GetKey(InstanceID, GetKey(BostionHostInstanceID))]
	if ok {
		if cluster.BostionHost == nil {
			cluster.BostionHost = &biz.BostionHost{}
		}
		cluster.BostionHost.InstanceID = bostionHostInstanceID
		for _, node := range cluster.Nodes {
			if node.InstanceID == bostionHostInstanceID {
				cluster.BostionHost.InternalIP = node.InternalIP
				cluster.BostionHost.ExternalIP = node.ExternalIP
				cluster.BostionHost.User = node.User
				break
			}
		}
	}
	return nil
}
