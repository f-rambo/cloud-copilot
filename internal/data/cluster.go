package data

import (
	"context"
	"fmt"
	"ocean/internal/biz"
	"os/exec"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/yaml.v3"
)

var clsuterKey = "cluster/config"

type clusterRepo struct {
	data *Data
	log  *log.Helper
}

func NewClusterRepo(data *Data, logger log.Logger) biz.ClusterRepo {
	return &clusterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// 获取集群配置
func (c *clusterRepo) GetCluster(ctx context.Context) (*biz.Cluster, error) {
	data, err := readFile(getClusterConfigPath())
	if err != nil {
		return nil, err
	}
	cluster := &biz.Cluster{}
	err = yaml.Unmarshal(data, cluster)
	return cluster, err
}

// 保存集群配置
func (c *clusterRepo) SaveCluster(ctx context.Context, cluster *biz.Cluster) error {
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}
	err = writeFile(getClusterConfigPath(), clusterData)
	if err != nil {
		return err
	}
	return nil
}

// 获取集群配置
// @param module 配置模块 cluster, addons, etcd
func (c *clusterRepo) GetClusterConfig(ctx context.Context, cluster *biz.Cluster, module string) ([]byte, error) {
	configPath, err := c.getConfigPath(ctx, cluster, module)
	if err != nil {
		return nil, err
	}
	return readFile(configPath)
}

// 保存集群配置
func (c *clusterRepo) SaveClusterConfig(ctx context.Context, cluster *biz.Cluster, module string, data []byte) error {
	configPath, err := c.getConfigPath(ctx, cluster, module)
	if err != nil {
		return err
	}
	return writeFile(configPath, data)
}

func (c *clusterRepo) getConfigPath(ctx context.Context, cluster *biz.Cluster, module string) (string, error) {
	// kubersparay 配置文件
	infraData, err := readFile(getInfraConfigPath())
	if err != nil {
		return "", err
	}
	var infra biz.Infra
	err = yaml.Unmarshal(infraData, &infra)
	if err != nil {
		return "", err
	}
	kubersparayPath := getKubersparayPath(infra.KubesprayVersion)
	var configPath string
	switch module {
	case "cluster":
		configPath = fmt.Sprintf("%s/inventory/%s/group_vars/k8s_cluster/k8s-cluster.yml", kubersparayPath, cluster.ClusterName)
	case "addons":
		configPath = fmt.Sprintf("%s/inventory/%s/group_vars/k8s_cluster/addons.yml", kubersparayPath, cluster.ClusterName)
	case "etcd":
		configPath = fmt.Sprintf("%s/inventory/%s/group_vars/etcd.yml", kubersparayPath, cluster.ClusterName)
	default:
	}
	if configPath == "" {
		return "", fmt.Errorf("config path not found")
	}
	return configPath, nil
}

func (c *clusterRepo) SetUpClusterTool(ctx context.Context, cluster *biz.Cluster) error {
	// 配置kubespray
	err := c.executeExpScript(ctx, "bash", getSetUpKubersparayPath(), getConfigPath())
	if err != nil {
		return err
	}
	// 设置服务器免密登录
	return c.executeExpScript(ctx, "bash", getSetServerLognPath(), getConfigPath())
}

// 部署集群
func (c *clusterRepo) DeployCluster(ctx context.Context, cluster *biz.Cluster) error {
	// 部署集群
	return c.executeExpScript(ctx, "bash", getSetUpClusterPath(), getConfigPath())
}

// 设置集群认证
func (c *clusterRepo) SetClusterAuth(ctx context.Context, cluster *biz.Cluster) error {
	var nodeName string
	for _, v := range cluster.Nodes {
		if containsValue(v.Role, "master") {
			nodeName = v.Name
			break
		}
	}
	if nodeName == "" {
		return fmt.Errorf("master node not found")
	}
	return c.executeExpScript(ctx, "bash", getCopyKubeConfigPath(), getConfigPath(), nodeName)
}

// 同步到etcd配置
func (c *clusterRepo) SyncConfigCluster(ctx context.Context) error {
	return nil
}

// 销毁集群
func (c *clusterRepo) DestroyCluster(ctx context.Context, cluster *biz.Cluster) error {
	return c.executeExpScript(ctx, "bash", getDestroyClusterPath(), getConfigPath())
}

// 添加节点
func (c *clusterRepo) AddNodes(ctx context.Context, cluster *biz.Cluster) error {
	return c.executeExpScript(ctx, "bash", getAddNodesPath(), getConfigPath())
}

// 删除节点
func (c *clusterRepo) RemoveNodes(ctx context.Context, nodes []string) error {
	nodesStr := strings.Join(nodes, ",")
	return c.executeExpScript(ctx, "bash", getRemoveNodesPath(), getConfigPath(), nodesStr)
}

func (c *clusterRepo) ClusterDataWatch(handler func(*biz.Cluster, *biz.Cluster) error) error {
	return nil
}

// 执行脚本
func (c *clusterRepo) executeExpScript(ctx context.Context, command string, scriptPath string, args ...string) error {
	// 创建 Cmd
	args = append([]string{scriptPath}, args...)
	cmd := exec.Command(command, args...)

	// 设置日志输出
	cmd.Stdout = c
	cmd.Stderr = c

	// 开始执行
	if err := cmd.Start(); err != nil {
		return err
	}

	// 等待命令执行完成
	if err := cmd.Wait(); err != nil {
		return err
	}
	c.log.Info(fmt.Sprintf("execute script %s success", scriptPath))
	return nil
}

// 实现 io.Writer 接口
func (c *clusterRepo) Write(bytes []byte) (int, error) {
	c.log.Info(string(bytes))
	return len(bytes), nil
}
