package kubernetes

import (
	"context"
	"encoding/json"
	"os"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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

func NewClusterRuntime(c *conf.Bootstrap, logger log.Logger) *ClusterRuntime {
	return &ClusterRuntime{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (cr *ClusterRuntime) createYAMLFile(ctx context.Context, dynamicClient *dynamic.DynamicClient, namespace, resource, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "open file failed")
	}
	defer file.Close()
	decoder := yaml.NewYAMLOrJSONDecoder(file, 1024)
	for {
		unstructuredObj := &unstructured.Unstructured{}
		if err := decoder.Decode(unstructuredObj); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return errors.Wrap(err, "decode yaml failed")
		}
		gvr := unstructuredObj.GroupVersionKind().GroupVersion().WithResource(resource)
		resourceClient := dynamicClient.Resource(gvr).Namespace(namespace)
		_, err = resourceClient.Create(ctx, unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create resource")
		}
	}
	return nil
}

func (cr *ClusterRuntime) CurrentCluster(ctx context.Context, cluster *biz.Cluster) (err error) {
	kubeClient, err := GetKubeClientByInCluster()
	if err != nil {
		return biz.ErrClusterNotFound
	}
	versionInfo, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	cluster.Version = versionInfo.String()
	err = cr.getClusterInfo(ctx, kubeClient, cluster)
	if err != nil {
		return err
	}
	err = cr.getNodes(ctx, kubeClient, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (cr *ClusterRuntime) InstallPlugins(ctx context.Context, cluster *biz.Cluster) error {
	kubeClient, err := GetDynamicClientByRestConfig(cluster.MasterIP, cluster.Token, cluster.CAData, cluster.KeyData, cluster.CertData)
	if err != nil {
		return err
	}
	caclicoYamls := []string{
		utils.MergePath(cr.c.Server.Install, "calico-operator.yaml"),
		utils.MergePath(cr.c.Server.Install, "calico.yaml"),
	}
	for _, v := range caclicoYamls {
		err = cr.createYAMLFile(ctx, kubeClient, "calico-system", "calico", v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cr *ClusterRuntime) DeployService(ctx context.Context, cluster *biz.Cluster) error {
	kubeClient, err := GetDynamicClientByRestConfig(cluster.MasterIP, cluster.Token, cluster.CAData, cluster.KeyData, cluster.CertData)
	if err != nil {
		return err
	}
	return cr.createYAMLFile(ctx, kubeClient, "kube-system", "ocean", utils.MergePath(cr.c.Server.Install, "ocean.yaml"))
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
