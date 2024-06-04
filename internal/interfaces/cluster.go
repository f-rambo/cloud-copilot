package interfaces

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/pkg/ansible"
	"github.com/f-rambo/ocean/pkg/pulumiapi"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	clusterUc *biz.ClusterUsecase
	projectUc *biz.ProjectUsecase
	appUc     *biz.AppUsecase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, projectUc *biz.ProjectUsecase, appUc *biz.AppUsecase, c *conf.Bootstrap, logger log.Logger) *ClusterInterface {
	return &ClusterInterface{
		clusterUc: clusterUc,
		projectUc: projectUc,
		appUc:     appUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
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
	bizCluster := c.clusterToBizCluster(cluster)
	err := c.clusterUc.Save(ctx, bizCluster)
	if err != nil {
		return nil, err
	}
	return c.bizCLusterToCluster(bizCluster), nil
}

func (c *ClusterInterface) List(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterList, error) {
	data := &v1alpha1.ClusterList{}
	clusters, err := c.clusterUc.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return data, nil
	}
	currentCluster, _ := c.clusterUc.CurrentCluster()
	for _, v := range clusters {
		cluster := c.bizCLusterToCluster(v)
		if currentCluster != nil && v.ApiServerAddress != "" && v.ApiServerAddress == currentCluster.ApiServerAddress {
			cluster.IsCurrentCluster = true
		}
		data.Clusters = append(data.Clusters, cluster)
	}
	return data, nil
}

func (c *ClusterInterface) GetCurrentCluster(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Cluster, error) {
	currentCluster, err := c.clusterUc.CurrentCluster()
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if currentCluster == nil {
		return data, nil
	}
	currentCluster.State = biz.ClusterStateRunning
	data = c.bizCLusterToCluster(currentCluster)
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

func (c *ClusterInterface) CheckClusterConfig(ctx context.Context, clusterId *v1alpha1.ClusterID) (*v1alpha1.Cluster, error) {
	if clusterId.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.CheckConfig(ctx, clusterId.Id)
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

func (c *ClusterInterface) SetUpCluster(ctx context.Context, clusterIdParam *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterIdParam.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.SetUpCluster(ctx, clusterIdParam.Id)
	if err != nil {
		return nil, err
	}
	cluster, err := c.clusterUc.Get(ctx, clusterIdParam.Id)
	if err != nil {
		return nil, err
	}
	err = c.appUc.BaseInstallation(ctx, cluster, nil)
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

// aws cluster deploy
func (c *ClusterInterface) AwsClusterDeploy(ctx context.Context, args *v1alpha1.AwsClusterDeployArgs) (*v1alpha1.Msg, error) {
	if args.AWS_ACCESS_KEY_ID == "" {
		return nil, errors.New("aws access key id is required")
	}
	if args.AWS_DEFAULT_REGION == "" {
		return nil, errors.New("aws default region is required")
	}
	if args.AWS_SECRET_ACCESS_KEY == "" {
		return nil, errors.New("aws secret access key is required")
	}
	if args.ClusterName == "" {
		return nil, errors.New("cluster name is required")
	}
	go func() {
		var err error
		defer func() {
			if err != nil {
				c.log.Error(err)
			}
		}()
		cluster := &biz.Cluster{
			Name:  args.ClusterName,
			Type:  "aws",
			Nodes: make([]*biz.Node, 0),
		}
		err = c.clusterUc.Save(ctx, cluster)
		if err != nil {
			return
		}
		nodeOptions := make([]pulumiapi.NodeGroupOptions, 0)
		for _, node := range args.NodeGroupOptions {
			nodeOptions = append(nodeOptions, pulumiapi.NodeGroupOptions{
				InstanceType: node.InstanceType,
				DesiredSize:  int(node.DesiredSize),
				MinSize:      int(node.MinSize),
				MaxSize:      int(node.MaxSize),
			})
		}

		g := new(errgroup.Group)
		pulumiOutput := make(chan string, 1024)
		g.Go(func() error {
			defer close(pulumiOutput)
			output, err := pulumiapi.NewPulumiAPI(ctx, pulumiOutput).ProjectName("aws-ocean").StackName("eks").Plugin("aws", "6.38.0").Env(map[string]string{
				"AWS_ACCESS_KEY_ID":     args.AWS_ACCESS_KEY_ID,
				"AWS_DEFAULT_REGION":    args.AWS_DEFAULT_REGION,
				"AWS_SECRET_ACCESS_KEY": args.AWS_SECRET_ACCESS_KEY,
			}).RegisterDeployFunc(pulumiapi.StartAwsEksCluster(pulumiapi.ClusterNodeGroupArgs{
				ClusterName:      args.ClusterName,
				NodeGroupName:    args.NodeGroupName,
				VPCID:            args.VpcId,
				SecurityGroupID:  args.SecurityGroupId,
				NodeGroupOptions: nodeOptions,
			})).Up(ctx)
			if err != nil {
				return err
			}
			c.log.Info("aws cluster deploy output:", output)
			ouputMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(output), &ouputMap)
			if err != nil {
				return err
			}
			kubeconfig, ok := ouputMap["kubeconfig"].(string)
			if !ok {
				err = errors.New("kubeconfig is not found")
				return err
			}
			cluster.KubeConfig = []byte(kubeconfig)
			return c.clusterUc.Save(ctx, cluster)
		})
		g.Go(func() error {
			return c.clusterUc.HandlerClusterLog(ctx, cluster, pulumiOutput)
		})
		err = g.Wait()
	}()
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) GetClusterMockData(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Cluster, error) {
	cresource := c.c.GetOceanResource()
	k, err := ansible.NewKubespray(&cresource)
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
	defaultClusterAddonsConfig, err := k.GetDefaultClusterAddonsConfig(ctx)
	if err != nil {
		return nil, err
	}
	cluster := &v1alpha1.Cluster{
		Name:             "cluster1",
		Config:           defaultClusterConfig,
		Addons:           defaultClusterAddons,
		AddonsConfig:     defaultClusterAddonsConfig,
		ApiServerAddress: "127.0.0.1:6443",
		Nodes: []*v1alpha1.Node{
			{
				Name:       "node1",
				ExternalIp: "192.168.90.130",
				User:       "username",
				Role:       "master",
			},
			{
				Name:         "node2",
				ExternalIp:   "192.168.90.132",
				User:         "username",
				Password:     "123456",
				SudoPassword: "123456",
				Role:         "worker",
			},
			{
				Name:       "node3",
				ExternalIp: "192.168.90.133",
				User:       "username",
				Role:       "worker",
			},
		},
	}
	return cluster, nil
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
