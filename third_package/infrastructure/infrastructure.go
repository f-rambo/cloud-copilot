package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	cloudv1alpha1 "github.com/f-rambo/ocean/api/cloud/v1alpha1"
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

const (
	TAG_KEY = "ocean-key"
	TAG_VAL = "ocean-cluster"

	VPC_STACK = "vpc-stack"

	PRIVATE_SUBNET_STACK = "private-subnet-stack-" // + zone
	PUBLIC_SUBNET_STACK  = "public-subnet-stack"

	INTERNETGATEWAY_STACK = "internetgateway-stack"

	PUBLIC_NATGATEWAY_EIP_STACK = "public-natgateway-eip-stack"
	BOSTIONHOST_EIP_STACK       = "bostionhost-eip-stack"

	PRIVATE_NATGATEWAY_STACK = "private-natgateway-stack" // + zone
	PUBLIC_NATGATEWAY_STACK  = "public-natgateway-stack"

	PUBLIC_NATGATEWAY_ROUTE_TABLE                  = "public-natgateway-route-table"
	PUBLIC__INTERNETGATEWAY_ROUTE_TABLE            = "public-internetgateway-route-table"
	PUBLIC_NATGATEWAY_ROUTE_TABLE_ASSOCIATION      = "public-natgateway-route-table-association"
	PUBLIC_INTERNETGATEWAY_ROUTE_TABLE_ASSOCIATION = "public-internetgateway-route-table-association"

	SECURITY_GROUP_STACK = "security-group-stack"

	EC2_ROLE_STACK         = "ec2-role-stack"
	EC2_ROLE_POLICY_STACK  = "ec2-role-policy-stack"
	EC2_ROLE_PROFILE_STACK = "ec2-role-profile-stack"

	EC2_INSTANCE_STACK = "ec2-instance-stack"

	KEY_PAIR_STACK = "key-pair-stack"

	BOSTIONHOST_STACK                   = "bostionhost-stack"
	BOSTIONHOST_NETWORK_INTERFACE_STACK = "bostionhost-network-interface-stack"
	BOSTIONHOST_EIP_ASSOCIATION_STACK   = "bostionhost-eip-association-stack"

	VPC_CIDR = "10.0.0.0/16"
)

const (
	// BOSTIONHOST
	BOSTIONHOST_EIP         = "bostionHostEip"
	BOSTIONHOST_INSTANCE_ID = "bostionHostInstanceId"
	BOSTIONHOST_PRIVATE_IP  = "bostionHostPrivateIp"
	BOSTIONHOST_USERNAME    = "bostionHostUsername"
)

const (
	PULUMI_ALICLOUD         = "alicloud"
	PULUMI_ALICLOUD_VERSION = "3.56.0"

	PULUMI_AWS         = "aws"
	PULUMI_AWS_VERSION = "6.38.0"

	PULUMI_KUBERNETES         = "kubernetes"
	PULUMI_KUBERNETES_VERSION = "4.12.0"
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

func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		_, err := c.pulumiExec(ctx, cluster, StartAlicloudCluster(cluster).StartServers)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		output, err := c.pulumiExec(ctx, cluster, StartEc2Instance(cluster).Start)
		if err != nil {
			return err
		}
		if output == "" {
			return nil
		}
		outputMap := make(map[string]interface{})
		json.Unmarshal([]byte(output), &outputMap)
		for k, v := range outputMap {
			if v == nil {
				continue
			}
			m := make(map[string]interface{})
			vJson, _ := json.Marshal(v)
			json.Unmarshal(vJson, &m)
			if _, ok := m["Value"]; !ok {
				continue
			}
			switch k {
			case BOSTIONHOST_EIP:
				cluster.BostionHost.ExternalIP = cast.ToString(m["Value"])
			case BOSTIONHOST_INSTANCE_ID:
				cluster.BostionHost.InstanceID = cast.ToString(m["Value"])
			case BOSTIONHOST_PRIVATE_IP:
				cluster.BostionHost.PrivateIP = cast.ToString(m["Value"])
			case BOSTIONHOST_USERNAME:
				cluster.BostionHost.Username = cast.ToString(m["Value"])
			}
		}
		return nil
	}
	return errors.New("not support cluster type")
}

func (c *ClusterInfrastructure) Stop(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		_, err := c.pulumiExec(ctx, cluster, StartAlicloudCluster(cluster).Clear)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		_, err := c.pulumiExec(ctx, cluster, StartEc2Instance(cluster).Clear)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("not support cluster type")
}

func (c *ClusterInfrastructure) Import(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		err := c.getNodesInformation(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		_, err := c.pulumiExec(ctx, cluster, StartAlicloudCluster(cluster).Import)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		_, err := c.pulumiExec(ctx, cluster, StartEc2Instance(cluster).Import)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("not support cluster type")
}

func (cc *ClusterInfrastructure) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	oceanAppVersion, err := utils.GetFromContextByKey(ctx, "version")
	if err != nil {
		return err
	}
	shipAppVersion, err := utils.GetFromContextByKey(ctx, "ship_version")
	if err != nil {
		return err
	}
	if cluster.BostionHost.Username == "" {
		return errors.New("bostion host username is empty")
	}
	if cluster.BostionHost.ExternalIP == "" {
		return errors.New("bostion host external ip is empty")
	}
	if cluster.BostionHost.SshPort == 0 {
		cluster.BostionHost.SshPort = 22
	}
	if cluster.BostionHost.ARCH == "" {
		output, err := exec.Command("uname", "-i").CombinedOutput()
		if err != nil {
			return errors.Wrap(err, string(output))
		}
		arch := strings.TrimSpace(string(output))
		if _, ok := ARCH_MAP[arch]; !ok {
			return errors.New("bostion host arch is not supported")
		}
		cluster.BostionHost.ARCH = ARCH_MAP[arch]
	}
	currentOceanFilePath, err := utils.GetPackageStorePathByNames()
	if err != nil {
		return err
	}
	output, err := exec.Command("tar", "-czvf", oceanDataTargzPackagePath, "-C", filepath.Dir(currentOceanFilePath), utils.PackageStoreDirName).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	// generating sha256sum
	output, err = exec.Command("sha256sum", oceanDataTargzPackagePath, ">", oceanDataTsha256sumFilePath).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	// scp to bostion host
	output, err = exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.SshPort), oceanDataTargzPackagePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, oceanDataTargzPackagePath)).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	// scp to bostion host
	output, err = exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.SshPort), oceanDataTsha256sumFilePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, oceanDataTsha256sumFilePath)).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	output, err = exec.Command("echo", installScript, "|", "ssh",
		fmt.Sprintf("%s@%s -p %d", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, cluster.BostionHost.SshPort),
		"sudo bash -s %s %s %s", cluster.BostionHost.ARCH, oceanAppVersion, shipAppVersion).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	return nil
}

func (cc *ClusterInfrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Status == biz.ClusterStatucCreating {
		err := cc.distributeShipServer(ctx, cluster)
		if err != nil {
			return err
		}
	}
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
			err := cc.downloadAndCopy(ctx, cluster, node)
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
			_, err = client.InitKubeadm(ctx, &cloudv1alpha1.Cloud{
				KubeadmInitConfig: utils.KubeadmInitConfig,
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

	return nil
}

func (cc *ClusterInfrastructure) AddNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {

	return nil
}

func (cc *ClusterInfrastructure) RemoveNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {

	return nil
}

func (cc *ClusterInfrastructure) GetServerEnv(context.Context) conf.Env {
	return cc.c.Server.GetEnv()
}

// Distribute the “ship server” to each node in the bostion host
func (cc *ClusterInfrastructure) distributeShipServer(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, _ := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
			if node.InternalIP == "" {
				return errors.New("node internal ip is empty")
			}
			if node.User == "" {
				return errors.New("node user is empty")
			}
			if node.SshPort == 0 {
				node.SshPort = 22
			}
			nodeGroup := cluster.GetNodeGroup(node.NodeGroupID)
			if nodeGroup == nil {
				nodeGroup = &biz.NodeGroup{}
			}
			if nodeGroup.ARCH == "" {
				output, err := exec.Command("echo", "uname -i", "|", "ssh", fmt.Sprintf("%s@%s -p %d", node.User, node.InternalIP, node.SshPort),
					"sudo bash -s").CombinedOutput()
				if err != nil {
					return errors.Wrap(err, string(output))
				}
				arch := strings.TrimSpace(string(output))
				if _, ok := ARCH_MAP[arch]; !ok {
					return errors.New("node arch is not supported")
				}
				nodeGroup.ARCH = ARCH_MAP[arch]
			}
			if nodeGroup.ARCH == "" {
				return errors.New("node arch is empty")
			}
			shipArchPath := fmt.Sprintf("%s/%s", shipPath, nodeGroup.ARCH)
			output, err := exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", node.SshPort), shipArchPath,
				fmt.Sprintf("%s@%s:%s", node.User, node.InternalIP, shipPath)).CombinedOutput()
			if err != nil {
				return errors.Wrap(err, string(output))
			}
			output, err = exec.Command("echo", shipStartScript, "|", "ssh",
				fmt.Sprintf("%s@%s -p %d", node.User, node.InternalIP, node.SshPort),
				"sudo bash -s %s", shipPath).CombinedOutput()
			if err != nil {
				return errors.Wrap(err, string(output))
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

func (cc *ClusterInfrastructure) getNodesInformation(ctx context.Context, cluster *biz.Cluster) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, node := range cluster.Nodes {
		node := node
		errGroup.Go(func() error {
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
			client := systemv1alpha1.NewSystemInterfaceClient(conn)
			appInfo := utils.GetFromContext(ctx)
			for k, v := range appInfo {
				ctx = metadata.AppendToClientContext(ctx, k, v)
			}
			systemInfo, err := client.GetSystem(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}
			node.SystemDisk = int32(systemInfo.DataDisk)
			node.GpuSpec = systemInfo.GpuSpec
			node.DataDisk = systemInfo.DataDisk
			node.Kernel = systemInfo.Kernel
			node.ContainerRuntime = systemInfo.Container
			node.Kubelet = systemInfo.Kubelet
			node.KubeProxy = systemInfo.KubeProxy
			node.InternalIP = systemInfo.InternalIp
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return err
	}
	return nil
}

// https://github.com/cri-o/cri-o/releases
func (c *ClusterInfrastructure) downloadAndCopy(_ context.Context, cluster *biz.Cluster, node *biz.Node) error {
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
	pulumiObj := NewPulumiAPI(ctx, cluster)
	if cluster.Type == biz.ClusterTypeAliCloud {
		pulumiObj.ProjectName(AlicloudProjectName).
			StackName(AlicloudStackName).
			Plugin([]PulumiPlugin{
				{Kind: PULUMI_ALICLOUD, Version: PULUMI_ALICLOUD_VERSION},
				{Kind: PULUMI_KUBERNETES, Version: PULUMI_KUBERNETES_VERSION},
			}...).
			Env(map[string]string{
				"ALICLOUD_ACCESS_KEY": cluster.AccessID,
				"ALICLOUD_SECRET_KEY": cluster.AccessKey,
				"ALICLOUD_REGION":     cluster.Region,
			})
	}
	if cluster.Type == biz.ClusterTypeAWS {
		pulumiObj.ProjectName(AWS_PROJECT).
			StackName(AWS_STACK).
			Plugin([]PulumiPlugin{
				{Kind: PULUMI_AWS, Version: PULUMI_AWS_VERSION},
				{Kind: PULUMI_KUBERNETES, Version: PULUMI_KUBERNETES_VERSION},
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
