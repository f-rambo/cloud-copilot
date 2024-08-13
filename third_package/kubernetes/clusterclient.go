package kubernetes

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	corev1 "k8s.io/api/core/v1"
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
		cr.log.Errorf("get kubeconfig error: %v", err)
		return biz.ErrClusterNotFound
	}
	cluster.ApiServerAddress = restConfig.Host
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		cr.log.Errorf("create clientset error: %v", err)
		return biz.ErrClusterNotFound
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
	err = cr.getNodes(ctx, clientSet, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (cr *ClusterRuntime) getClusterInfo(ctx context.Context, clientSet *kubernetes.Clientset, cluster *biz.Cluster) error {
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

func (cr *ClusterRuntime) getNodes(ctx context.Context, clientSet *kubernetes.Clientset, cluster *biz.Cluster) error {
	nodeRes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodeRes.Items {
		n := &biz.Node{}
		clusterNodeIndex := -1
		for index, v := range cluster.Nodes {
			if v.Name == node.Name {
				n = v
				clusterNodeIndex = index
				break
			}
		}
		n.Name = node.Name
		for _, v := range node.Status.Addresses {
			if v.Address == "" {
				continue
			}
			if v.Type == "InternalIP" {
				n.InternalIP = v.Address
			}
			if v.Type == "ExternalIP" {
				n.ExternalIP = v.Address
			}
		}
		n.Kubelet = node.Status.NodeInfo.KubeletVersion
		n.Container = node.Status.NodeInfo.ContainerRuntimeVersion
		n.Kernel = node.Status.NodeInfo.KernelVersion
		n.KubeProxy = node.Status.NodeInfo.KubeProxyVersion
		n.Status = biz.NodeStatusUnspecified
		for _, v := range node.Status.Conditions {
			if v.Status == corev1.ConditionStatus(corev1.NodeReady) {
				n.Status = biz.NodeStatusRunning
			}
		}
		nodeLables, err := json.Marshal(node)
		if err != nil {
			return err
		}
		err = json.Unmarshal(nodeLables, &n)
		if err != nil {
			return err
		}
		n.Labels = string(nodeLables)
		if clusterNodeIndex == -1 {
			cluster.Nodes = append(cluster.Nodes, n)
		} else {
			cluster.Nodes[clusterNodeIndex] = n
		}
	}
	return nil
}
