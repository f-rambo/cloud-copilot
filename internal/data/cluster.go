package data

import (
	"context"
	"encoding/json"
	"fmt"
	"ocean/internal/biz"
	"ocean/internal/data/restapi"
	"strings"

	"ocean/utils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type kubesprayRepo struct {
	Name                    string `yaml:"name"`
	Repository              string `yaml:"repository"`
	Branch                  string `yaml:"branch"`
	ClusterConfig           string `yaml:"cluster_config"`
	AddonsConfig            string `yaml:"addons_config"`
	DeployCluster           string `yaml:"deploy_cluster"`
	AddNode                 string `yaml:"addnode"`
	RemoveNode              string `yaml:"removenode"`
	Reset                   string `yaml:"reset"`
	ClusterInit             string `yaml:"cluster_init"`
	DeployKubeedgeEdgedCore string `yaml:"kubeedge_edged"`
	DeployKubeedgeCloudCore string `yaml:"kubeedge_cloud"`
	KubeedgeReset           string `yaml:"kubeedge_cloud"`
	KeyType                 string `yaml:"key_type"`
	NormalUserKey           string `yaml:"normal_user_key"`
	RootUserKey             string `yaml:"root_user_key"`
}

var repo kubesprayRepo

func init() {
	repo = kubesprayRepo{
		"kubespray",
		"https://github.com/f-rambo/kubespray.git",
		"master",
		"https://raw.githubusercontent.com/f-rambo/kubespray/master/inventory/sample/group_vars/k8s_cluster/k8s-cluster.yml",
		"https://raw.githubusercontent.com/f-rambo/kubespray/master/inventory/sample/group_vars/k8s_cluster/addons.yml",
		"cluster.yml",
		"scale.yml",
		"remove-node.yml",
		"reset.yml",
		"cluster-init.yml",
		"kubeedge-edged.yml",
		"kubeedge-cloudcore.yml",
		"kubeedge-reset.yml",
		"none",
		"normal_user_key",
		"root_user_key",
	}
}

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

func (c *clusterRepo) SaveCluster(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.Applyed {
		err := c.deleteCacheCluster(ctx, cluster.Name)
		if err != nil {
			return err
		}
		return c.saveToDB(ctx, cluster)
	}
	// 创建项目
	err := c.saveProject(ctx, cluster)
	if err != nil {
		return err
	}
	// 创建秘钥对
	err = c.saveKey(ctx, cluster)
	if err != nil {
		return err
	}
	// 创建git仓库
	err = c.saveRepositories(ctx, cluster)
	if err != nil {
		return err
	}
	// 环境变量
	err = c.saveEnvironment(ctx, cluster)
	if err != nil {
		return err
	}
	// 主机配置
	err = c.saveInventory(ctx, cluster)
	if err != nil {
		return err
	}
	// 创建模版
	// 集群初始化
	err = c.saveTemplate(ctx, cluster, repo.ClusterInit, "", "集群初始化", nil)
	if err != nil {
		return err
	}
	// 部署集群
	err = c.saveTemplate(ctx, cluster, repo.DeployCluster, "", "部署集群", nil)
	if err != nil {
		return err
	}
	// 卸载集群
	err = c.saveTemplate(ctx, cluster, repo.Reset, "", "卸载集群", nil)
	if err != nil {
		return err
	}
	// 添加节点
	err = c.saveTemplate(ctx, cluster, repo.AddNode, "", "添加节点", nil)
	if err != nil {
		return err
	}
	// 移除节点
	err = c.saveTemplate(ctx, cluster, repo.RemoveNode, "", "移除节点", nil)
	if err != nil {
		return err
	}
	// 卸载kubeedge边缘节点
	err = c.saveTemplate(ctx, cluster, repo.KubeedgeReset, "", "卸载kubeedge边缘节点", nil)
	if err != nil {
		return err
	}
	// kubedge edged core
	err = c.saveTemplate(ctx, cluster, repo.DeployKubeedgeEdgedCore, "", "部署边缘节点", nil)
	if err != nil {
		return err
	}
	// todo kubeedge
	return c.saveToCache(ctx, cluster)
}

func (c *clusterRepo) saveToDB(ctx context.Context, cluster *biz.Cluster) (err error) {
	// 写入数据库
	return c.data.db.Transaction(func(tx *gorm.DB) error {
		// 写入cluste
		config, err := utils.YamlToJson(string(cluster.Config))
		if err != nil {
			return err
		}
		cluster.Config = []byte(config)
		addons, err := utils.YamlToJson(string(cluster.Addons))
		if err != nil {
			return err
		}
		cluster.Addons = []byte(addons)
		if err = tx.Save(cluster).Error; err != nil {
			return err
		}
		nodes := make([]biz.Node, 0)
		err = tx.Where("cluster_id = ?", cluster.ID).Find(&nodes).Error
		if err != nil {
			return err
		}
		delNodeIDs := make([]int, 0)
		for _, node := range nodes {
			isExist := false
			for _, n := range cluster.Nodes {
				if n.ID == node.ID {
					isExist = true
					break
				}
			}
			if !isExist {
				delNodeIDs = append(delNodeIDs, node.ID)
			}
		}
		if len(delNodeIDs) > 0 {
			err = tx.Where("id in ?", delNodeIDs).Delete(biz.Node{}).Error
			if err != nil {
				return err
			}
		}
		for _, node := range cluster.Nodes {
			rj, err := json.Marshal(node.Role)
			if err != nil {
				return err
			}
			node.ClusterID = cluster.ID
			node.RoleJson = string(rj)
			if cluster.CreatedAt.Unix() > 0 {
				node.CreatedAt = cluster.CreatedAt
			}
			if err = tx.Save(&node).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *clusterRepo) saveToCache(ctx context.Context, cluster *biz.Cluster) error {
	cluster.ConfigStr = string(cluster.Config)
	cluster.AddonsStr = string(cluster.Addons)
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	err = c.data.redisClient.Set(cluster.Name, clusterJson, 0).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	err = c.addClusterPool(ctx, cluster.Name)
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}

func (c *clusterRepo) getCacheCluster(ctx context.Context, clusterName string) (*biz.Cluster, error) {
	clusterJson, err := c.data.redisClient.Get(clusterName).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if err == redis.Nil {
		return nil, nil
	}
	cluster := &biz.Cluster{}
	err = json.Unmarshal([]byte(clusterJson), cluster)
	if err != nil {
		return nil, err
	}
	cluster.Addons = []byte(cluster.AddonsStr)
	cluster.Config = []byte(cluster.ConfigStr)
	return cluster, nil
}

func (c *clusterRepo) deleteCacheCluster(ctx context.Context, clusterName string) error {
	err := c.data.redisClient.Del(clusterName).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	err = c.deleteClusterPool(ctx, clusterName)
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}

func (c *clusterRepo) addClusterPool(ctx context.Context, clusterName string) error {
	return c.data.redisClient.SAdd("cluster_pool", clusterName).Err()
}

func (c *clusterRepo) deleteClusterPool(ctx context.Context, clusterName string) error {
	err := c.data.redisClient.SRem("cluster_pool", clusterName).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}

func (c *clusterRepo) getClusterPool(ctx context.Context) ([]string, error) {
	clusternames, err := c.data.redisClient.SMembers("cluster_pool").Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	return clusternames, nil
}

func (c *clusterRepo) GetClusterByName(ctx context.Context, clusterName string) (*biz.Cluster, error) {
	clusterCache, err := c.getCacheCluster(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	if clusterCache != nil {
		return clusterCache, nil
	}
	cluster := &biz.Cluster{}
	err = c.data.db.Where("cluster_name = ?", clusterName).First(cluster).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return cluster, nil
}

func (c *clusterRepo) GetCluster(ctx context.Context, id int) (*biz.Cluster, error) {
	if id == 0 {
		return nil, nil
	}
	cluster := &biz.Cluster{}
	err := c.data.db.First(cluster, id).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]biz.Node, 0)
	err = c.data.db.Find(&nodes, "cluster_id = ?", cluster.ID).Error
	if err != nil {
		return nil, err
	}
	cluster.Nodes = nodes
	return cluster, err
}

func (c *clusterRepo) GetClusters(ctx context.Context) ([]*biz.Cluster, error) {
	clusters := make([]*biz.Cluster, 0)
	err := c.data.db.Find(&clusters).Error
	if err != nil {
		return nil, err
	}
	clusterIds := make([]int, 0)
	for _, v := range clusters {
		clusterIds = append(clusterIds, v.ID)
	}
	nodes := make([]biz.Node, 0)
	err = c.data.db.Where("cluster_id in ?", clusterIds).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	for _, v := range clusters {
		for _, n := range nodes {
			if v.ID == n.ClusterID {
				var role []string
				err = json.Unmarshal([]byte(n.RoleJson), &role)
				if err != nil {
					return nil, err
				}
				n.Role = role
				v.Nodes = append(v.Nodes, n)
				break
			}
		}
	}
	// cache
	clusterNames, err := c.getClusterPool(ctx)
	if err != nil {
		return nil, err
	}
	for _, name := range clusterNames {
		v, err := c.getCacheCluster(ctx, name)
		if err != nil {
			return nil, err
		}
		if v == nil {
			continue
		}
		v.Addons = []byte(v.AddonsStr)
		v.Config = []byte(v.ConfigStr)
		clusters = append(clusters, v)
	}
	return clusters, nil
}

func (c *clusterRepo) DeleteCluster(ctx context.Context, cluster *biz.Cluster) error {
	// 删除项目
	semaphore := c.data.semaphore
	tms, err := semaphore.GetTemplates(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	for _, v := range tms {
		err = semaphore.DeleteTemplate(cluster.SemaphoreID, v.ID)
		if err != nil {
			return err
		}
	}
	// host
	hosts, err := semaphore.GetInventorys(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	for _, v := range hosts {
		err = semaphore.DeleteInventory(cluster.SemaphoreID, v.ID)
		if err != nil {
			return err
		}
	}
	// env
	envs, err := semaphore.GetEnvironments(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	for _, v := range envs {
		err = semaphore.DeleteEnvironments(cluster.SemaphoreID, v.ID)
		if err != nil {
			return err
		}
	}
	// repo
	repos, err := semaphore.GetRepositories(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	for _, v := range repos {
		err = semaphore.DeleteRepositories(cluster.SemaphoreID, v.ID)
		if err != nil {
			return err
		}
	}
	// key
	keys, err := semaphore.GetKeys(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	for _, v := range keys {
		err = semaphore.DeleteKey(cluster.SemaphoreID, v.ID)
		if err != nil {
			return err
		}
	}
	// project
	err = semaphore.DeleteProject(cluster.SemaphoreID)
	if err != nil {
		return err
	}
	// 写入数据库
	return c.data.db.Transaction(func(tx *gorm.DB) error {
		err = tx.Delete(cluster).Error
		if err != nil {
			return err
		}
		err = tx.Where("cluster_id = ?", cluster.ID).Delete(&biz.Node{}).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func (c *clusterRepo) ClusterInit(ctx context.Context, cluster *biz.Cluster) error {
	// 获取模版
	templateID, ok := cluster.TemplateIDs[repo.ClusterInit]
	if !ok {
		return fmt.Errorf("ClusterInit template not found")
	}
	task := &restapi.Task{
		TemplateID: cast.ToInt(templateID),
		ProjectId:  cluster.SemaphoreID,
		Debug:      true,
		Diff:       true,
	}
	err := c.data.semaphore.StartTask(cluster.SemaphoreID, task)
	if err != nil {
		return err
	}
	cluster.SetTaskID(repo.ClusterInit, task.ID)
	return nil
}

func (c *clusterRepo) DeployCluster(ctx context.Context, cluster *biz.Cluster) error {
	// 获取模版
	templateID, ok := cluster.TemplateIDs[repo.DeployCluster]
	if !ok {
		return fmt.Errorf("DeployCluster template not found")
	}
	task := &restapi.Task{
		TemplateID: cast.ToInt(templateID),
		ProjectId:  cluster.SemaphoreID,
		Debug:      true,
		Diff:       true,
	}
	err := c.data.semaphore.StartTask(cluster.SemaphoreID, task)
	if err != nil {
		return err
	}
	cluster.SetTaskID(repo.DeployCluster, task.ID)
	return nil
}

func (c *clusterRepo) UndeployCluster(ctx context.Context, cluster *biz.Cluster) error {
	// 获取模版
	templateID, ok := cluster.TemplateIDs[repo.Reset]
	if !ok {
		return fmt.Errorf("UndeployCluster template not found")
	}
	task := &restapi.Task{
		TemplateID: cast.ToInt(templateID),
		ProjectId:  cluster.SemaphoreID,
		Debug:      true,
		Diff:       true,
	}
	err := c.data.semaphore.StartTask(cluster.SemaphoreID, task)
	if err != nil {
		return err
	}
	cluster.SetTaskID(repo.Reset, task.ID)
	return nil
}

func (c *clusterRepo) AddNode(ctx context.Context, cluster *biz.Cluster) error {
	// 获取模版
	templateID, ok := cluster.TemplateIDs[repo.AddNode]
	if !ok {
		return fmt.Errorf("AddNode template not found")
	}
	task := &restapi.Task{
		TemplateID: cast.ToInt(templateID),
		ProjectId:  cluster.SemaphoreID,
		Debug:      true,
		Diff:       true,
	}
	err := c.data.semaphore.StartTask(cluster.SemaphoreID, task)
	if err != nil {
		return err
	}
	cluster.SetTaskID(repo.AddNode, task.ID)
	return nil
}

func (c *clusterRepo) RemoveNode(ctx context.Context, cluster *biz.Cluster, nodes []biz.Node) error {
	// 获取模版
	templateID, ok := cluster.TemplateIDs[repo.RemoveNode]
	if !ok {
		return fmt.Errorf("RemoveNode template not found")
	}
	nodeNames := make([]string, 0)
	for _, v := range nodes {
		nodeNames = append(nodeNames, v.Name)
	}
	task := &restapi.Task{
		TemplateID: cast.ToInt(templateID),
		ProjectId:  cluster.SemaphoreID,
		Debug:      true,
		Diff:       true,
		Arguments:  fmt.Sprintf(`["--extra-vars", "node=%s"]`, strings.Join(nodeNames, ",")),
	}
	err := c.data.semaphore.StartTask(cluster.SemaphoreID, task)
	if err != nil {
		return err
	}
	cluster.SetTaskID(repo.RemoveNode, task.ID)
	return nil
}

func (c *clusterRepo) AddKubeEdge(ctx context.Context, cluster *biz.Cluster) error {
	// todo
	return nil
}

func (c *clusterRepo) RemoveKubeEdge(ctx context.Context, cluster *biz.Cluster) error {
	// todo
	return nil
}

func (c *clusterRepo) saveProject(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.SemaphoreID != 0 {
		return nil
	}
	project := restapi.Project{Name: cluster.Name, Alert: true}
	err := c.data.semaphore.CreateProject(&project)
	if err != nil {
		return err
	}
	cluster.SetSemaphoreID(project.ID)
	return nil
}

func (c *clusterRepo) saveKey(ctx context.Context, cluster *biz.Cluster) error {
	key := restapi.Key{
		ID:        cluster.KeyID,
		Name:      cluster.Name,
		Type:      repo.KeyType,
		ProjectID: cluster.SemaphoreID,
	}
	if cluster.KeyID != 0 {
		return c.data.semaphore.UpdateKey(key)
	}
	err := c.data.semaphore.CreateKey(&key)
	if err != nil {
		return err
	}
	cluster.SetKeyID(key.ID)
	return nil
}

func (c *clusterRepo) saveRepositories(ctx context.Context, cluster *biz.Cluster) error {
	repoData := restapi.Repository{
		ID:        cluster.RepoID,
		Name:      repo.Name,
		ProjectID: cluster.SemaphoreID,
		GitURL:    repo.Repository,
		GitBranch: repo.Branch,
		SSHKeyID:  cluster.KeyID,
	}
	if cluster.RepoID != 0 {
		return c.data.semaphore.UpdateRepositories(&repoData)
	}
	err := c.data.semaphore.CreateRepositories(&repoData)
	if err != nil {
		return err
	}
	cluster.SetRepoID(repoData.ID)
	return nil
}

func (c *clusterRepo) saveEnvironment(ctx context.Context, cluster *biz.Cluster) error {
	jsonStr, err := utils.YamlToJson(string(cluster.Config), string(cluster.Addons))
	if err != nil {
		return err
	}
	env := restapi.Environment{
		ID:        cluster.EnvID,
		Name:      repo.Name,
		ProjectID: cluster.SemaphoreID,
		JSON:      jsonStr,
	}
	if cluster.EnvID != 0 {
		return c.data.semaphore.UpdateEnvironments(env)
	}
	err = c.data.semaphore.CreateEnvironment(&env)
	if err != nil {
		return err
	}
	cluster.SetEnvID(env.ID)
	return nil
}

func (c *clusterRepo) GetDefaultCluster(ctx context.Context) (*biz.Cluster, error) {
	config, err := restapi.GetContentByUrl(repo.ClusterConfig)
	if err != nil {
		return nil, err
	}
	addons, err := restapi.GetContentByUrl(repo.AddonsConfig)
	if err != nil {
		return nil, err
	}
	node := biz.Node{
		Name:         "node1",
		Host:         "x.x.x.x",
		User:         "root",
		Password:     "root",
		SudoPassword: "root",
		Role:         []string{"master", "worker"},
	}
	cluster := &biz.Cluster{
		Name:  "k8s-cluster",
		Nodes: []biz.Node{node},
	}
	cluster.Config = []byte(config)
	cluster.Addons = []byte(addons)
	return cluster, nil
}

func (c *clusterRepo) saveInventory(ctx context.Context, cluster *biz.Cluster) error {
	inventoryType, inventoryFile := c.getInventoryFile(cluster)
	inventory := restapi.Inventory{
		ID:        cluster.InventoryID,
		Name:      cluster.Name,
		ProjectID: cluster.SemaphoreID,
		Type:      inventoryType,
		SSHKeyID:  cluster.KeyID,
		BecomeKey: cluster.KeyID,
		Inventory: inventoryFile,
	}
	if cluster.InventoryID != 0 {
		return c.data.semaphore.UpdateInventory(inventory)
	}
	err := c.data.semaphore.CreateInventory(&inventory)
	if err != nil {
		return err
	}
	cluster.SetInventoryID(inventory.ID)
	return nil
}

func (c *clusterRepo) saveTemplate(ctx context.Context, cluster *biz.Cluster, playBook, arguments, description string, surveyVars []restapi.SurveyVar) error {
	template := restapi.Template{
		ProjectID:             cluster.SemaphoreID,
		Name:                  repo.Name + "-" + playBook,
		Playbook:              playBook,
		Arguments:             arguments,
		Description:           description,
		SuppressSuccessAlerts: true,
		SurveyVars:            surveyVars,
		Inventory:             cluster.InventoryID,
		Repository:            cluster.RepoID,
		Environment:           cluster.EnvID,
	}
	if id, ok := cluster.TemplateIDs[playBook]; ok {
		template.ID = cast.ToInt(id)
		return c.data.semaphore.UpdateTemplate(template)
	}
	err := c.data.semaphore.CreateTemplate(&template)
	if err != nil {
		return err
	}
	cluster.SetTemplateIDs(playBook, template.ID)
	return nil
}

func (c *clusterRepo) getInventoryFile(cluster *biz.Cluster) (string, string) {
	lines := make([]string, 0)
	lines = append(lines, "[all]")
	etcdNum := 1
	for _, node := range cluster.Nodes {
		name := node.Name
		host := node.Host
		user := node.User
		password := node.Password
		sudoPassword := node.SudoPassword
		role := node.Role

		line := fmt.Sprintf("%s ansible_host=%s ip=%s access_ip=%s ansible_user=%s ansible_password=%s ansible_become_password=%s", name, host, host, host, user, password, sudoPassword)
		if utils.Contains(role, "master") {
			line += fmt.Sprintf(" etcd_member_name=etcd%d", etcdNum)
			etcdNum++
			lines = append(lines, line)
			continue
		}
		lines = append(lines, line)
	}

	lines = append(lines, "", "[kube_control_plane]")
	for _, node := range cluster.Nodes {
		if utils.Contains(node.Role, "master") {
			lines = append(lines, node.Name)
		}
	}

	lines = append(lines, "", "[etcd]")
	for _, node := range cluster.Nodes {
		if utils.Contains(node.Role, "master") {
			lines = append(lines, node.Name)
		}
	}

	lines = append(lines, "", "[kube_node]")
	for _, node := range cluster.Nodes {
		if utils.Contains(node.Role, "worker") {
			lines = append(lines, node.Name)
		}
	}

	lines = append(lines,
		"",
		"[calico_rr]",
		"",
		"[k8s_cluster:children]",
		"kube_control_plane",
		"kube_node",
		"calico_rr",
		"")
	return "static", strings.Join(lines, "\n")
}
