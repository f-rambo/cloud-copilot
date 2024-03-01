package interfaces

import (
	"context"

	"github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/ansible"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/release"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	clusterUc *biz.ClusterUsecase
	projectUc *biz.ProjectUsecase
	appUc     *biz.AppUsecase
	c         *conf.Resource
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, projectUc *biz.ProjectUsecase, appUc *biz.AppUsecase, c *conf.Resource, logger log.Logger) (*ClusterInterface, error) {
	cluster := &ClusterInterface{
		clusterUc: clusterUc,
		projectUc: projectUc,
		appUc:     appUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
	ctx := context.Background()
	bizCluster, err := cluster.clusterInit(ctx)
	if err != nil {
		cluster.log.Errorf("not cluster error: %v", err)
		return cluster, nil
	}
	bizProject, err := cluster.ProjectInit(ctx, bizCluster)
	if err != nil {
		return nil, err
	}
	err = cluster.AppInit(ctx, bizCluster, bizProject)
	return cluster, err
}

func (c *ClusterInterface) clusterInit(ctx context.Context) (*biz.Cluster, error) {
	cluster, err := c.clusterUc.CurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	clusters, err := c.clusterUc.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range clusters {
		if v.Name == cluster.Name {
			cluster.ID = v.ID
			break
		}
	}
	err = c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *ClusterInterface) ProjectInit(ctx context.Context, cluster *biz.Cluster) (*biz.Project, error) {
	projects, err := c.projectUc.List(ctx, cluster.ID)
	if err != nil {
		return nil, err
	}
	for _, v := range projects {
		if v.Name == cluster.Name {
			return v, nil
		}
	}
	project := &biz.Project{
		Name:      cluster.Name,
		Namespace: cluster.Name,
		ClusterID: cluster.ID,
	}
	err = c.projectUc.Save(ctx, project)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (c *ClusterInterface) AppInit(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	appinstallfuncs := []func(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error{
		c.installOpenEBS,
		c.installPrometheus,
		c.installGrafana,
		c.installHarbor,
		c.installTraefik,
		c.installIstio,
	}
	for _, appinstallfunc := range appinstallfuncs {
		err := appinstallfunc(ctx, cluster, project)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterInterface) installOpenEBS(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	repo := &biz.AppHelmRepo{
		Name: "openebs",
		Url:  "https://openebs.github.io/charts",
	}
	err := c.appUc.SaveRepo(ctx, repo)
	if err != nil {
		return err
	}
	var appName string = "openebs"
	var version string = "3.10.0"
	deployApps, _, err := c.appUc.DeployAppList(ctx,
		biz.DeployApp{
			RepoID: repo.ID, AppName: appName, Version: version, AppTypeID: biz.AppTypeRepo, ClusterID: cluster.ID, ProjectID: project.ID,
		},
		1, 1)
	if err != nil {
		return err
	}
	var deployAppID int64 = 0
	for _, deployApp := range deployApps {
		deployAppID = deployApp.ID
		if deployApp.State == release.StatusDeployed.String() {
			return nil
		}
	}
	configmap := map[string]interface{}{}
	yamlByte, err := yaml.Marshal(configmap)
	if err != nil {
		return err
	}
	deployApp := &biz.DeployApp{
		ID:        deployAppID,
		ClusterID: cluster.ID,
		ProjectID: project.ID,
		AppName:   appName,
		AppTypeID: biz.AppTypeRepo,
		RepoID:    repo.ID,
		Version:   version,
		UserID:    biz.AdminID,
		Config:    string(yamlByte),
	}
	_, err = c.appUc.DeployApp(ctx, deployApp)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterInterface) installPrometheus(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	return nil
}

func (c *ClusterInterface) installGrafana(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	return nil
}

func (c *ClusterInterface) installHarbor(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	return nil
}

func (c *ClusterInterface) installTraefik(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	return nil
}

func (c *ClusterInterface) installIstio(ctx context.Context, cluster *biz.Cluster, project *biz.Project) error {
	return nil
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Msg, error) {
	return &v1alpha1.Msg{Message: "pong"}, nil
}

func (c *ClusterInterface) Get(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Cluster, error) {
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	data = c.bizCLusterToCluster(cluster)
	return data, nil
}

func (c *ClusterInterface) Save(ctx context.Context, cluster *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	if cluster.Name == "" {
		return nil, errors.New("cluster name is required")
	}
	if cluster.ApiServerAddress == "" {
		return nil, errors.New("api server address is required")
	}
	bizCluster := c.clusterToBizCluster(cluster)
	err := c.clusterUc.Save(ctx, bizCluster)
	if err != nil {
		return nil, err
	}
	return c.bizCLusterToCluster(bizCluster), nil
}

func (c *ClusterInterface) List(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterList, error) {
	clusters, err := c.clusterUc.List(ctx)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.ClusterList{}
	for _, v := range clusters {
		data.Clusters = append(data.Clusters, c.bizCLusterToCluster(v))
	}
	return data, nil
}

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) DeleteNode(ctx context.Context, clusterParam *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterParam.Id == 0 || clusterParam.NodeId == 0 {
		return nil, errors.New("cluster id is required and node id is required")
	}
	err := c.clusterUc.DeleteNode(ctx, clusterParam.Id, clusterParam.NodeId)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) GetClusterMockData(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Cluster, error) {
	k, err := ansible.NewKubespray(c.c)
	if err != nil {
		return nil, err
	}
	defaultClusterConfig, err := k.GetDefaultClusterConfig(ctx)
	if err != nil {
		return nil, err
	}
	defaultClusterAddons, err := k.GetDefaultClusterAddons(ctx)
	if err != nil {
		return nil, err
	}
	cluster := &v1alpha1.Cluster{
		Name:             "cluster1",
		Config:           defaultClusterConfig,
		Addons:           defaultClusterAddons,
		ApiServerAddress: "127.0.0.1:6443",
		Nodes: []*v1alpha1.Node{
			{
				Name:         "node1",
				ExternalIp:   "127.0.0.1:22",
				User:         "root",
				Password:     "password",
				SudoPassword: "sudo password",
				Role:         "master/worker/edge",
			},
		},
	}
	return cluster, nil
}

func (c *ClusterInterface) SetUpCluster(ctx context.Context, cluster *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if cluster.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.SetUpCluster(ctx, cluster.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

// UninstallCluster
func (c *ClusterInterface) UninstallCluster(ctx context.Context, cluster *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if cluster.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.UninstallCluster(ctx, cluster.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

// AddNode
func (c *ClusterInterface) AddNode(ctx context.Context, clusterParam *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterParam.Id == 0 || clusterParam.NodeId == 0 {
		return nil, errors.New("cluster id is required and node id is required")
	}
	err := c.clusterUc.AddNode(ctx, clusterParam.Id, clusterParam.NodeId)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

// RemoveNode
func (c *ClusterInterface) RemoveNode(ctx context.Context, clusterParam *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterParam.Id == 0 || clusterParam.NodeId == 0 {
		return nil, errors.New("cluster id is required and node id is required")
	}
	err := c.clusterUc.RemoveNode(ctx, clusterParam.Id, clusterParam.NodeId)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) bizCLusterToCluster(bizCluster *biz.Cluster) *v1alpha1.Cluster {
	cluster := &v1alpha1.Cluster{
		Id:               bizCluster.ID,
		Name:             bizCluster.Name,
		ServerVersion:    bizCluster.ServerVersion,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Config:           bizCluster.Config,
		Addons:           bizCluster.Addons,
		State:            bizCluster.State,
		Logs:             bizCluster.Logs,
	}
	for _, node := range bizCluster.Nodes {
		cluster.Nodes = append(cluster.Nodes, c.bizNodeToNode(node))
	}
	return cluster
}

func (c *ClusterInterface) bizNodeToNode(bizNode *biz.Node) *v1alpha1.Node {
	node := &v1alpha1.Node{
		Id:           bizNode.ID,
		Name:         bizNode.Name,
		Labels:       bizNode.Labels,
		Annotations:  bizNode.Annotations,
		OsImage:      bizNode.OSImage,
		Kernel:       bizNode.Kernel,
		Container:    bizNode.Container,
		Kubelet:      bizNode.Kubelet,
		KubeProxy:    bizNode.KubeProxy,
		InternalIp:   bizNode.InternalIP,
		ExternalIp:   bizNode.ExternalIP,
		User:         bizNode.User,
		Password:     bizNode.Password,
		SudoPassword: bizNode.SudoPassword,
		Role:         bizNode.Role,
		State:        bizNode.State,
		ClusterId:    bizNode.ClusterID,
	}
	return node
}

func (c *ClusterInterface) clusterToBizCluster(cluster *v1alpha1.Cluster) *biz.Cluster {
	bizCluster := &biz.Cluster{
		ID:               cluster.Id,
		Name:             cluster.Name,
		ServerVersion:    cluster.ServerVersion,
		ApiServerAddress: cluster.ApiServerAddress,
		Config:           cluster.Config,
		Addons:           cluster.Addons,
	}
	for _, node := range cluster.Nodes {
		bizCluster.Nodes = append(bizCluster.Nodes, c.nodeToBizNode(node))
	}
	return bizCluster
}

func (c *ClusterInterface) nodeToBizNode(node *v1alpha1.Node) *biz.Node {
	bizNode := &biz.Node{
		ID:           node.Id,
		Name:         node.Name,
		Labels:       node.Labels,
		Annotations:  node.Annotations,
		OSImage:      node.OsImage,
		Kernel:       node.Kernel,
		Container:    node.Container,
		Kubelet:      node.Kubelet,
		KubeProxy:    node.KubeProxy,
		InternalIP:   node.InternalIp,
		ExternalIP:   node.ExternalIp,
		User:         node.User,
		Password:     node.Password,
		SudoPassword: node.SudoPassword,
		Role:         node.Role,
		ClusterID:    node.ClusterId,
	}
	return bizNode
}
