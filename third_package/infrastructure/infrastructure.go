package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

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
		SetPlugin(PulumiPlugin{Kind: "alicloud", Version: "3.56.0"}, PulumiPlugin{Kind: "kubernetes", Version: "4.12.0"}).
		SetEnv("ALICLOUD_ACCESS_KEY", cluster.AccessID).
		SetEnv("ALICLOUD_SECRET_KEY", cluster.AccessKey).
		SetEnv("ALICLOUD_REGION", cluster.Region)
}

func (c *ClusterInfrastructure) buildAwsCloudParam(cluster *biz.Cluster) {
	c.SetProjectName(AWS_PROJECT).
		SetStackName(AWS_STACK).
		SetPlugin(PulumiPlugin{Kind: "aws", Version: "6.38.0"}, PulumiPlugin{Kind: "kubernetes", Version: "4.12.0"}).
		SetEnv("AWS_ACCESS_KEY_ID", cluster.AccessID).
		SetEnv("AWS_SECRET_ACCESS_KEY", cluster.AccessKey).
		SetEnv("AWS_DEFAULT_REGION", cluster.Region)
}

// 在云厂商创建服务器
func (c *ClusterInfrastructure) Start(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		c.buildAliCloudParam(cluster)
		_, log, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).StartServers)
		if err != nil {
			return err
		}
		cluster.Logs = log
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		c.buildAwsCloudParam(cluster)
		_, log, err := c.pulumiExec(ctx, StartEc2Instance(cluster).Start)
		if err != nil {
			return err
		}
		cluster.Logs = log
	}
	return errors.New("not support cluster type")
}

// 删除云厂商服务器
func (c *ClusterInfrastructure) Stop(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		c.buildAliCloudParam(cluster)
		_, log, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).Clear)
		if err != nil {
			return err
		}
		cluster.Logs = log
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		c.buildAwsCloudParam(cluster)
		_, log, err := c.pulumiExec(ctx, StartEc2Instance(cluster).Clear)
		if err != nil {
			return err
		}
		cluster.Logs = log
	}
	return nil
}

// 获取当前集群的服务器信息，如果未安装则返回 not found
func (c *ClusterInfrastructure) Get(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.GetType() == biz.ClusterTypeLocal {
		return nil
	}
	if cluster.GetType() == biz.ClusterTypeAliCloud {
		c.buildAliCloudParam(cluster)
		_, log, err := c.pulumiExec(ctx, StartAlicloudCluster(cluster).Get)
		if err != nil {
			return err
		}
		cluster.Logs = log
	}
	if cluster.GetType() == biz.ClusterTypeAWS {
		ec2InstanceObj := StartEc2Instance(cluster)
		c.buildAwsCloudParam(cluster)
		output, log, err := c.pulumiExec(ctx, ec2InstanceObj.Get)
		if err != nil {
			return err
		}
		cluster.Logs = log
		err = ec2InstanceObj.DecodeClusterInfomation(cluster, output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterInfrastructure) pulumiExec(ctx context.Context, pulumiFunc PulumiFunc) (output, processLog string, err error) {
	g := new(errgroup.Group)
	pulumiProcessLog := make(chan string, 1024)
	g.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("pulumi error: %s", err)
			}
			close(pulumiProcessLog)
		}()
		output, err = NewPulumiAPI(ctx, pulumiProcessLog).
			ProjectName(c.projectName).
			StackName(c.stack).
			Plugin(c.plugins...).
			Env(c.env).
			RegisterDeployFunc(pulumiFunc).
			Up(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		for {
			select {
			case log, ok := <-pulumiProcessLog:
				if !ok {
					return nil
				}
				processLog += log
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	err = g.Wait()
	return output, processLog, err
}

// 初始化集群
func (cc *ClusterInfrastructure) GenerateInitial(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Type == biz.ClusterTypeLocal && len(cluster.Nodes) < biz.NodeMinSize.Int() {
		return errors.New("local cluster node size must be greater than 1")
	}
	// 云厂商集群
	if cluster.Type != biz.ClusterTypeLocal {
		nodeGroup := &biz.NodeGroup{
			CPU:                     4,
			Memory:                  8,
			SystemDisk:              100,
			InternetMaxBandwidthOut: 100,
			MinSize:                 2,
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
	// 本地集群
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

// 获取集群节点信息，配置信息
func (cc *ClusterInfrastructure) getNodesInformation(ctx context.Context, cluster *biz.Cluster) error {
	playbook := getSystemInformation()
	playbookPath, err := savePlaybook(cc.c.Resource.GetClusterPath(), playbook)
	if err != nil {
		return err
	}
	err = cc.ansibleExec(ctx, cluster, cc.c.Resource.GetClusterPath(), playbookPath, cc.generatingNodes(cluster))
	if err != nil {
		return err
	}
	resultMaps := make([]map[string]interface{}, 0)
	remaining := cluster.Logs
	for {
		startIndex := strings.Index(remaining, StartOutputKey.String())
		if startIndex == -1 {
			break
		}
		endIndex := strings.Index(remaining, EndOutputKey.String())
		if endIndex == -1 {
			break
		}
		startIndex += len(StartOutputKey.String())
		if startIndex >= endIndex {
			break
		}
		result := remaining[startIndex:endIndex]
		if result != "" {
			unescapedResult := strings.ReplaceAll(result, `\"`, `"`)
			resultMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(unescapedResult), &resultMap)
			if err != nil {
				return err
			}
			resultMaps = append(resultMaps, resultMap)
		}
		remaining = remaining[endIndex+len(EndOutputKey.String()):]
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
				nodeGroup.OSImage = cast.ToString(v)
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

func (cc *ClusterInfrastructure) GenerateNodeLables(ctx context.Context, cluster *biz.Cluster, nodeGroup *biz.NodeGroup) (lables string, err error) {
	lableMap := make(map[string]string)
	lableMap["cluster"] = cluster.Name
	lableMap["cluster_type"] = cluster.Type.String()
	lableMap["region"] = cluster.Region
	lableMap["nodegroup"] = nodeGroup.Name
	lableMap["nodegroup_type"] = nodeGroup.Type.String()
	lableMap["instance_type"] = nodeGroup.InstanceType
	lablebytes, err := json.Marshal(lableMap)
	if err != nil {
		return "", err
	}
	return string(lablebytes), nil
}

func (cc *ClusterInfrastructure) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	defer func() {
		// 迁移完成后，关闭服务
		ctx.Done()
	}()
	oceanResource := cc.c.Resource
	migratePlaybook := getMigratePlaybook()
	databasePath := cc.c.Data.GetDBFilePath()
	pulumiPath := cc.c.Resource.GetPulumiPath()
	migratePlaybook.AddSynchronize("database", databasePath, databasePath)
	migratePlaybook.AddSynchronize("pulumi", pulumiPath, pulumiPath)
	migratePlaybookPath, err := savePlaybook(oceanResource.GetClusterPath(), migratePlaybook)
	if err != nil {
		return err
	}
	err = cc.ansibleExec(ctx, cluster, oceanResource.GetClusterPath(), migratePlaybookPath, cc.generatingBostionHost(cluster))
	if err != nil {
		return err
	}
	// 把本地的数据迁移到bostion主机
	return nil
}

func (cc *ClusterInfrastructure) Install(ctx context.Context, cluster *biz.Cluster) error {
	serversInitPlaybook := getServerInitPlaybook()
	serversInitPlaybookPath, err := savePlaybook(cc.c.Resource.GetClusterPath(), serversInitPlaybook)
	if err != nil {
		return err
	}
	err = cc.ansibleExec(ctx, cluster, cc.c.Resource.GetClusterPath(), serversInitPlaybookPath, cc.generatingNodes(cluster))
	if err != nil {
		return err
	}
	return cc.kubespray(ctx, cluster, GetClusterPlaybookPath())
}

func (cc *ClusterInfrastructure) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	return cc.kubespray(ctx, cluster, GetResetPlaybookPath())
}

func (cc *ClusterInfrastructure) AddNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("add node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
	}
	return cc.kubespray(ctx, cluster, GetScalePlaybookPath())
}

func (cc *ClusterInfrastructure) RemoveNodes(ctx context.Context, cluster *biz.Cluster, nodes []*biz.Node) error {
	for _, node := range nodes {
		log.Info("remove node", "name", node.Name, "ip", node.ExternalIP, "role", node.Role)
		err := cc.kubespray(ctx, cluster, GetRemoveNodePlaybookPath(), map[string]string{"node": node.Name})
		if err != nil {
			return err
		}
	}
	return nil
}

func (cc *ClusterInfrastructure) kubespray(ctx context.Context, cluster *biz.Cluster, playbook string, env ...map[string]string) error {
	oceanResource := cc.c.Resource
	kubespray, err := NewKubespray(&oceanResource)
	if err != nil {
		return errors.Wrap(err, "new kubespray error")
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

func (cc *ClusterInfrastructure) generatingBostionHost(cluster *biz.Cluster) []Server {
	return []Server{
		{Ip: cluster.BostionHost.ExternalIP, Username: "root", ID: cluster.BostionHost.InstanceID, Role: "bostion"},
	}
}

func (cc *ClusterInfrastructure) generatingNodes(cluster *biz.Cluster) []Server {
	servers := make([]Server, 0)
	for _, node := range cluster.Nodes {
		servers = append(servers, Server{Ip: node.ExternalIP, Username: node.User, ID: cast.ToString(node.ID), Role: node.Role.String()})
	}
	return servers
}

func (cc *ClusterInfrastructure) ansibleExec(ctx context.Context, cluster *biz.Cluster, cmdRunDir string, playbook string, servers []Server, envAndMateData ...map[string]string) error {
	env := make(map[string]string)
	mateData := make(map[string]string)
	if len(envAndMateData) > 0 && envAndMateData[0] != nil {
		env = envAndMateData[0]
	}
	if len(envAndMateData) > 1 && envAndMateData[1] != nil {
		mateData = envAndMateData[1]
	}
	g := new(errgroup.Group)
	ansibleLog := make(chan string, 1024)
	g.Go(func() error {
		defer close(ansibleLog)
		return NewGoAnsiblePkg(cc.c.Ansible).
			SetAnsiblePlaybookBinary(cc.c.Resource.GetAnsibleCli()).
			SetLogChan(ansibleLog).
			SetServers(servers...).
			SetCmdRunDir(cmdRunDir).
			SetPlaybooks(playbook).
			SetMatedataMap(mateData).
			SetEnvMap(env).
			ExecPlayBooks(ctx)
	})
	g.Go(func() error {
		for {
			select {
			case log, ok := <-ansibleLog:
				if !ok {
					return nil
				}
				cluster.Logs += log
				cc.log.Info(log)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	err := g.Wait()
	if err != nil {
		return err
	}
	return nil
}
