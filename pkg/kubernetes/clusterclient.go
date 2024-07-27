package kubernetes

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type CLusterRuntimeConfigMapKey string

func (k CLusterRuntimeConfigMapKey) String() string {
	return string(k)
}

const (
	ClusterInformation   CLusterRuntimeConfigMapKey = "cluster-info"
	NodegroupInformation CLusterRuntimeConfigMapKey = "nodegroup-info"
	NodeLableKey         CLusterRuntimeConfigMapKey = "node-lable"
)

type ClusterRuntime struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewClusterRuntime(c *conf.Bootstrap, logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntime{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (cr *ClusterRuntime) CurrentCluster(ctx context.Context, cluster *biz.Cluster) error {
	if cluster.KubeConfig == "" {
		exist := utils.IsFileExist(clientcmd.RecommendedHomeFile)
		if exist {
			cluster.KubeConfig = clientcmd.RecommendedHomeFile
		}
	}
	if cluster.KubeConfig == "" {
		config, err := clientcmd.LoadFromFile(cluster.KubeConfig)
		if err != nil {
			return err
		}
		cluster.Name = config.CurrentContext
	}
	restConfig, err := getKubeConfig(&ConfigArgs{KubeConfig: cluster.KubeConfig})
	if err != nil {
		return err
	}
	cluster.ApiServerAddress = restConfig.Host
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	// get cluster information kubectl cluster-info dump
	versionInfo, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	cluster.ServerVersion = versionInfo.String()
	err = cr.getClusterInfo(ctx, clientSet, cluster)
	if err != nil {
		return err
	}
	err = cr.getNodeGroupInfo(ctx, clientSet, cluster)
	if err != nil {
		return err
	}
	err = cr.getNodes(ctx, clientSet, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (cr *ClusterRuntime) getClusterInfo(ctx context.Context, clientSet *kubernetes.Clientset, cluster *biz.Cluster) error {
	// cluster infomation in configmap
	configMap, err := clientSet.CoreV1().ConfigMaps("kube-system").Get(ctx, ClusterInformation.String(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if _, ok := configMap.Data[ClusterInformation.String()]; !ok {
		return nil
	}
	err = json.Unmarshal([]byte(configMap.Data[ClusterInformation.String()]), cluster)
	if err != nil {
		return err
	}
	return nil
}

func (cr *ClusterRuntime) getNodeGroupInfo(ctx context.Context, clientSet *kubernetes.Clientset, cluster *biz.Cluster) error {
	// nodegroup infomation in configmap
	configMap, err := clientSet.CoreV1().ConfigMaps("kube-system").Get(ctx, NodegroupInformation.String(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if _, ok := configMap.Data[NodegroupInformation.String()]; !ok {
		return nil
	}
	nodegroups := make([]*biz.NodeGroup, 0)
	err = json.Unmarshal([]byte(configMap.Data[NodegroupInformation.String()]), &nodegroups)
	if err != nil {
		return err
	}
	cluster.NodeGroups = nodegroups
	return nil
}

func (cr *ClusterRuntime) getNodes(ctx context.Context, clientSet *kubernetes.Clientset, cluster *biz.Cluster) error {
	nodes := make([]*biz.Node, 0)
	nodeRes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodeRes.Items {
		n := &biz.Node{}
		nodeLables, err := json.Marshal(node)
		if err != nil {
			return err
		}
		err = json.Unmarshal(nodeLables, &n)
		if err != nil {
			return err
		}
		if n.Name == "" {
			n.Name = node.Name
		}
		n.Labels = string(nodeLables)
		nodes = append(nodes, n)
	}
	cluster.Nodes = nodes
	return nil
}
