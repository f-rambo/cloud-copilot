package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
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
	oceanAppVersion, shipAppVersion, err := utils.GetAppVersionFromContext(ctx)
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
		err = cc.generateInitial(ctx, cluster)
		if err != nil {
			return err
		}
	}
	//...
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
			arch := node.NodeGroup.ARCH
			if arch == "" {
				output, err := exec.Command("echo", "uname -i", "|", "ssh", fmt.Sprintf("%s@%s -p %d", node.User, node.InternalIP, node.SshPort),
					"sudo bash -s").CombinedOutput()
				if err != nil {
					return errors.Wrap(err, string(output))
				}
				arch = strings.TrimSpace(string(output))
				if _, ok := ARCH_MAP[arch]; !ok {
					return errors.New("node arch is not supported")
				}
				node.NodeGroup.ARCH = ARCH_MAP[arch]
			}
			if node.NodeGroup.ARCH == "" {
				return errors.New("node arch is empty")
			}
			shipArchPath := fmt.Sprintf("%s/%s", shipPath, node.NodeGroup.ARCH)
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

func (cc *ClusterInfrastructure) generateInitial(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Type == biz.ClusterTypeLocal && len(cluster.Nodes) < biz.NodeMinSize.Int() {
		return errors.New("local cluster node size must be greater than 1")
	}
	if cluster.Type != biz.ClusterTypeLocal {
		return cc.generateInitialCloud(ctx, cluster)
	}
	return cc.generateInitialLocal(ctx, cluster)
}

func (cc *ClusterInfrastructure) generateInitialCloud(_ context.Context, cluster *biz.Cluster) error {
	nodeGroup := &biz.NodeGroup{
		CPU:                     2,
		Memory:                  4,
		SystemDisk:              100,
		InternetMaxBandwidthOut: 100,
		MinSize:                 5,
		TargetSize:              5,
	}
	nodeGroup.Name = fmt.Sprintf("cloudproider-%s-cpu-%d-mem-%d-disk-%d", cluster.Type, nodeGroup.CPU, int(nodeGroup.Memory), nodeGroup.SystemDisk)
	cluster.NodeGroups = []*biz.NodeGroup{nodeGroup}
	var i int32
	for i = 0; i < nodeGroup.MinSize; i++ {
		roleName := biz.NodeRoleMaster.String()
		if i > 2 {
			roleName = biz.NodeRoleWorker.String()
		}
		node := &biz.Node{
			Name:      fmt.Sprintf("%s-%d", roleName, i),
			Labels:    "",
			Status:    biz.NodeStatusRunning,
			ClusterID: cluster.ID,
			NodeGroup: nodeGroup,
			Role:      biz.NodeRole(roleName),
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}
	cluster.BostionHost = &biz.BostionHost{
		Memory: 4,
		CPU:    2,
	}
	return nil
}

func (cc *ClusterInfrastructure) generateInitialLocal(ctx context.Context, cluster *biz.Cluster) error {
	err := cc.getNodesInformation(ctx, cluster)
	if err != nil {
		return err
	}
	sort.Sort(biz.Nodes(cluster.Nodes))
	masterNum := 0
	workNum := 0
	for _, node := range cluster.Nodes {
		if node.NodeGroup == nil {
			return errors.New("node group is nil")
		}
		if node.NodeGroup.Memory >= 8 && node.NodeGroup.CPU >= 4 && masterNum < 3 {
			node.Role = biz.NodeRoleMaster
			node.Status = biz.NodeStatusCreating
			masterNum++
			continue
		}
		if workNum >= 3 {
			node.Status = biz.NodeStatusUnspecified
			continue
		}
		node.Role = biz.NodeRoleWorker
		node.Status = biz.NodeStatusCreating
		workNum++
	}
	return nil
}

func (cc *ClusterInfrastructure) getNodesInformation(_ context.Context, cluster *biz.Cluster) error {
	for _, node := range cluster.Nodes {
		fmt.Println(node)
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
