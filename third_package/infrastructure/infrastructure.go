package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	log         *log.Helper
	c           *conf.Bootstrap
	projectName string
	stack       string
	plugins     []PulumiPlugin
	env         map[string]string
}

func NewClusterInfrastructure(c *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &ClusterInfrastructure{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (c *ClusterInfrastructure) SetProjectName(projectName string) *ClusterInfrastructure {
	c.projectName = projectName
	return c
}

func (c *ClusterInfrastructure) SetStackName(stackName string) *ClusterInfrastructure {
	c.stack = stackName
	return c
}

func (c *ClusterInfrastructure) SetPlugin(plugins ...PulumiPlugin) *ClusterInfrastructure {
	if c.plugins == nil {
		c.plugins = make([]PulumiPlugin, 0)
	}
	c.plugins = append(c.plugins, plugins...)
	return c
}

func (c *ClusterInfrastructure) SetEnv(key, val string) *ClusterInfrastructure {
	if c.env == nil {
		c.env = make(map[string]string)
	}
	c.env[key] = val
	return c
}

func (c *ClusterInfrastructure) buildAliCloudParam(cluster *biz.Cluster) {
	c.SetProjectName(AlicloudProjectName).
		SetStackName(AlicloudStackName).
		SetPlugin(PulumiPlugin{Kind: PULUMI_ALICLOUD, Version: PULUMI_ALICLOUD_VERSION}, PulumiPlugin{Kind: PULUMI_KUBERNETES, Version: PULUMI_KUBERNETES_VERSION}).
		SetEnv("ALICLOUD_ACCESS_KEY", cluster.AccessID).
		SetEnv("ALICLOUD_SECRET_KEY", cluster.AccessKey).
		SetEnv("ALICLOUD_REGION", cluster.Region)
}

func (c *ClusterInfrastructure) buildAwsCloudParam(cluster *biz.Cluster) {
	c.SetProjectName(AWS_PROJECT).
		SetStackName(AWS_STACK).
		SetPlugin(PulumiPlugin{Kind: PULUMI_AWS, Version: PULUMI_AWS_VERSION}, PulumiPlugin{Kind: PULUMI_KUBERNETES, Version: PULUMI_KUBERNETES_VERSION}).
		SetEnv("AWS_ACCESS_KEY_ID", cluster.AccessID).
		SetEnv("AWS_SECRET_ACCESS_KEY", cluster.AccessKey).
		SetEnv("AWS_DEFAULT_REGION", cluster.Region)
}

func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		c.buildAliCloudParam(cluster)
		_, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).StartServers, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		c.buildAwsCloudParam(cluster)
		output, err := c.pulumiExec(ctx, StartEc2Instance(cluster).Start, cluster)
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
		c.buildAliCloudParam(cluster)
		_, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).Clear, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		c.buildAwsCloudParam(cluster)
		_, err := c.pulumiExec(ctx, StartEc2Instance(cluster).Clear, cluster)
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
		c.buildAliCloudParam(cluster)
		_, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).Import, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		ec2InstanceObj := StartEc2Instance(cluster)
		c.buildAwsCloudParam(cluster)
		_, err := c.pulumiExec(ctx, ec2InstanceObj.Import, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("not support cluster type")
}

// preview is true, only preview the pulumi resources
func (c *ClusterInfrastructure) pulumiExec(ctx context.Context, pulumiFunc PulumiFunc, w io.Writer, preview ...bool) (output string, err error) {
	pulumiObj := NewPulumiAPI(ctx, w).
		ProjectName(c.projectName).
		StackName(c.stack).
		Plugin(c.plugins...).
		Env(c.env).
		RegisterDeployFunc(pulumiFunc)
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
	if cluster.BostionHost.Port == 0 {
		cluster.BostionHost.Port = 22
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
	output, err = exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.Port), oceanDataTargzPackagePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, oceanDataTargzPackagePath)).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	// scp to bostion host
	output, err = exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", cluster.BostionHost.Port), oceanDataTsha256sumFilePath,
		fmt.Sprintf("%s@%s:%s", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, oceanDataTsha256sumFilePath)).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(output))
	}
	output, err = exec.Command("echo", installScript, "|", "ssh",
		fmt.Sprintf("%s@%s -p %d", cluster.BostionHost.Username, cluster.BostionHost.ExternalIP, cluster.BostionHost.Port),
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
	serversInitPlaybook := getServerInitPlaybook()
	clusterPath, err := utils.GetPackageStorePathByNames(biz.ClusterPackageName)
	if err != nil {
		return err
	}
	serversInitPlaybookPath, err := savePlaybook(clusterPath, serversInitPlaybook)
	if err != nil {
		return err
	}
	_, err = cc.ansibleExec(ctx, cluster, clusterPath, serversInitPlaybookPath, cc.generatingNodes(cluster))
	if err != nil {
		return err
	}
	_, err = cc.kubespray(ctx, cluster, GetClusterPlaybookPath())
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	_, err := cc.kubespray(ctx, cluster, GetResetPlaybookPath())
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) AddNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("add node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
	}
	_, err := cc.kubespray(ctx, cluster, GetScalePlaybookPath())
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClusterInfrastructure) RemoveNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("remove node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
		_, err := cc.kubespray(ctx, cluster, GetRemoveNodePlaybookPath(), map[string]string{"node": node.Name})
		if err != nil {
			return err
		}
	}
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
			if node.Port == 0 {
				node.Port = 22
			}
			arch := node.NodeGroup.ARCH
			if arch == "" {
				output, err := exec.Command("echo", "uname -i", "|", "ssh", fmt.Sprintf("%s@%s -p %d", node.User, node.InternalIP, node.Port),
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
			output, err := exec.Command("scp", "-r", "-P", fmt.Sprintf("%d", node.Port), shipArchPath,
				fmt.Sprintf("%s@%s:%s", node.User, node.InternalIP, shipPath)).CombinedOutput()
			if err != nil {
				return errors.Wrap(err, string(output))
			}
			output, err = exec.Command("echo", shipStartScript, "|", "ssh",
				fmt.Sprintf("%s@%s -p %d", node.User, node.InternalIP, node.Port),
				"sudo bash -s %s", shipPath).CombinedOutput()
			if err != nil {
				return errors.Wrap(err, string(output))
			}
			// grpc connect....
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

func (cc *ClusterInfrastructure) getNodesInformation(ctx context.Context, cluster *biz.Cluster) error {
	playbook := getSystemInformation()
	clusterPath, err := utils.GetPackageStorePathByNames(biz.ClusterPackageName)
	if err != nil {
		return err
	}
	playbookPath, err := savePlaybook(clusterPath, playbook)
	if err != nil {
		return err
	}
	output, err := cc.ansibleExec(ctx, cluster, clusterPath, playbookPath, cc.generatingNodes(cluster))
	if err != nil {
		return err
	}
	resultMaps := make([]map[string]interface{}, 0)
	for {
		startIndex := strings.Index(output, StartOutputKey.String())
		if startIndex == -1 {
			break
		}
		endIndex := strings.Index(output, EndOutputKey.String())
		if endIndex == -1 {
			break
		}
		startIndex += len(StartOutputKey.String())
		if startIndex >= endIndex {
			break
		}
		result := output[startIndex:endIndex]
		if result != "" {
			unescapedResult := strings.ReplaceAll(result, `\"`, `"`)
			resultMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(unescapedResult), &resultMap)
			if err != nil {
				return err
			}
			resultMaps = append(resultMaps, resultMap)
		}
		output = output[endIndex+len(EndOutputKey.String()):]
	}
	getNodeResult := func(nodeID int64) map[string]interface{} {
		for _, resultMap := range resultMaps {
			if _, ok := resultMap["node_id"]; ok && cast.ToInt64(resultMap["node_id"]) == nodeID {
				return resultMap
			}
		}
		return nil
	}
	for _, node := range cluster.Nodes {
		resultMap := getNodeResult(node.ID)
		nodeGroup := &biz.NodeGroup{}
		for k, v := range resultMap {
			switch k {
			case "gpu_number":
				nodeGroup.GPU = cast.ToInt32(v)
			case "gpu_spec":
				nodeGroup.GpuSpec = cast.ToString(v)
			case "cpu_number":
				nodeGroup.CPU = cast.ToInt32(v)
			case "memory":
				nodeGroup.Memory = cast.ToFloat64(v)
			case "disk":
				nodeGroup.SystemDisk = cast.ToInt32(v)
			case "os_info":
				nodeGroup.OS = cast.ToString(v)
			}
		}
		node.NodeGroup = nodeGroup
		node.Kernel = cast.ToString(resultMap["kernel_info"])
		node.Container = cast.ToString(resultMap["container_version"])
		node.InternalIP = cast.ToString(resultMap["ip"])
	}
	nodeGroupMap := make(map[string]*biz.NodeGroup)
	for _, node := range cluster.Nodes {
		nodeGroupName := fmt.Sprintf("gpu-%d-gpu_spec-%s-cpu-%d-mem-%d-disk-%d",
			node.NodeGroup.GPU, node.NodeGroup.GpuSpec, node.NodeGroup.CPU, int(node.NodeGroup.Memory), node.NodeGroup.DataDisk)
		node.NodeGroup.Name = nodeGroupName
		nodeGroupMap[nodeGroupName] = node.NodeGroup
	}
	nodeGroups := make([]*biz.NodeGroup, 0)
	for _, nodeGroup := range nodeGroupMap {
		nodeGroups = append(nodeGroups, nodeGroup)
	}
	cluster.NodeGroups = nodeGroups
	return nil
}

func (cc *ClusterInfrastructure) kubespray(ctx context.Context, cluster *biz.Cluster, playbook string, env ...map[string]string) (string, error) {
	kubespray, err := NewKubespray()
	if err != nil {
		return "", errors.Wrap(err, "new kubespray error")
	}
	mateDataMap := make(map[string]string)
	for _, node := range cluster.Nodes {
		mateDataMap[node.Name] = node.ExternalIP
	}
	if len(env) > 0 && env[0] != nil {
		return cc.ansibleExec(ctx, cluster, kubespray.GetPackagePath(), playbook, cc.generatingNodes(cluster), env[0], mateDataMap)
	}
	return cc.ansibleExec(ctx, cluster, kubespray.GetPackagePath(), playbook, cc.generatingNodes(cluster), nil, mateDataMap)
}

func (cc *ClusterInfrastructure) generatingNodes(cluster *biz.Cluster) []Server {
	servers := make([]Server, 0)
	for _, node := range cluster.Nodes {
		servers = append(servers, Server{Ip: node.ExternalIP, Username: node.User, ID: cast.ToString(node.ID), Role: node.Role.String()})
	}
	return servers
}

func (cc *ClusterInfrastructure) ansibleExec(ctx context.Context, cluster *biz.Cluster, cmdRunDir string, playbook string, servers []Server, envAndMateData ...map[string]string) (string, error) {
	env := make(map[string]string)
	mateData := make(map[string]string)
	if len(envAndMateData) > 0 && envAndMateData[0] != nil {
		env = envAndMateData[0]
	}
	if len(envAndMateData) > 1 && envAndMateData[1] != nil {
		mateData = envAndMateData[1]
	}
	ansibleObj, err := NewGoAnsiblePkg(cluster)
	if err != nil {
		return "", err
	}
	output, err := ansibleObj.
		SetServers(servers...).
		SetCmdRunDir(cmdRunDir).
		SetPlaybooks(playbook).
		SetMatedataMap(mateData).
		SetEnvMap(env).
		ExecPlayBooks(ctx)
	if err != nil {
		return "", err
	}
	return output, nil
}
