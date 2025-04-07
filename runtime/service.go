package runtime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	CloudServiceKind = "CloudService"
)

type ServiceRuntime struct {
	log *log.Helper
}

func NewServiceRuntime(logger log.Logger) biz.ServiceRuntime {
	return &ServiceRuntime{
		log: log.NewHelper(logger),
	}
}

type CloudServiceType string

const (
	CloudServiceTypeHttpServer CloudServiceType = "HttpServer"
	CloudServiceTypeGrpcServer CloudServiceType = "GrpcServer"
)

// CloudServiceSpec defines the desired state of CloudService.
type CloudServiceSpec struct {
	CloudServiceType     CloudServiceType  `json:"cloud_service_type,omitempty"`
	Gateway              string            `json:"gateway,omitempty"`
	Image                string            `json:"image,omitempty"`
	Replicas             int32             `json:"replicas,omitempty"`
	RequestCPU           int32             `json:"request_cpu,omitempty"`
	LimitCPU             int32             `json:"limit_cpu,omitempty"`
	RequestGPU           int32             `json:"request_gpu,omitempty"`
	LimitGPU             int32             `json:"limit_gpu,omitempty"`
	RequestMemory        int32             `json:"request_memory,omitempty"`
	LimitMemory          int32             `json:"limit_memory,omitempty"`
	Volumes              []Volume          `json:"volumes,omitempty"`
	Ports                []Port            `json:"ports,omitempty"`
	ConfigPath           string            `json:"config_path,omitempty"` // dir
	Config               map[string]string `json:"config,omitempty"`      // key: filename, value: content
	IngressNetworkPolicy []NetworkPolicy   `json:"ingress_network_policy,omitempty"`
	EgressNetworkPolicy  []NetworkPolicy   `json:"egress_network_policy,omitempty"`
	CanaryDeployment     CanaryDeployment  `json:"canary_deployment,omitempty"`
}

type Port struct {
	Name          string `json:"name,omitempty"`
	IngressPath   string `json:"ingress_path,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	ContainerPort int32  `json:"container_port,omitempty"`
}

type Volume struct {
	Name         string `json:"name,omitempty"`
	Path         string `json:"path,omitempty"`
	Storage      int32  `json:"storage,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
}

type CanaryDeployment struct {
	Image      string            `json:"image,omitempty"`
	Replicas   int32             `json:"replicas,omitempty"`
	Config     map[string]string `json:"config,omitempty"` // key: filename, value: content
	TrafficPct int32             `json:"traffic_pct,omitempty"`
}

type NetworkPolicy struct {
	IpCIDR      string            `json:"ip_cidr,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	MatchLabels map[string]string `json:"match_labels,omitempty"`
}

func (s *ServiceRuntime) ApplyService(ctx context.Context, service *biz.Service, continuousDeployment *biz.ContinuousDeployment) error {
	cluster := biz.GetCluster(ctx)
	workspace := biz.GetWorkspace(ctx)
	project := biz.GetProject(ctx)
	cloudServiceSpec := &CloudServiceSpec{
		CloudServiceType: CloudServiceTypeHttpServer,
		Gateway:          service.Gateway,
		Image:            continuousDeployment.Image,
		Replicas:         service.Replicas,
		RequestCPU:       service.RequestCpu,
		LimitCPU:         service.LimitCpu,
		RequestGPU:       service.RequestGpu,
		LimitGPU:         service.LimitGpu,
		RequestMemory:    service.RequestMemory,
		Config:           continuousDeployment.Config,
	}
	for _, v := range service.Volumes {
		cloudServiceSpec.Volumes = append(cloudServiceSpec.Volumes, Volume{
			Name:         v.Name,
			Path:         v.MountPath,
			Storage:      v.Storage,
			StorageClass: v.StorageClass,
		})
	}
	for _, v := range service.Ports {
		cloudServiceSpec.Ports = append(cloudServiceSpec.Ports, Port{
			Name:          v.Name,
			ContainerPort: v.ContainerPort,
			Protocol:      v.Protocol,
			IngressPath:   v.IngressPath,
		})
	}
	cloudServiceSpec.IngressNetworkPolicy = []NetworkPolicy{
		{
			Namespace:   service.Namespace,
			MatchLabels: utils.MergeMaps(cluster.GetLabels(), workspace.GetLabels(), project.GetLabels()),
		},
	}
	if continuousDeployment.IsAccessExternal == biz.AccessExternal_False {
		cloudServiceSpec.EgressNetworkPolicy = []NetworkPolicy{
			{
				Namespace:   service.Namespace,
				MatchLabels: utils.MergeMaps(cluster.GetLabels(), workspace.GetLabels(), project.GetLabels()),
			},
		}
	}
	if continuousDeployment.CanaryDeployment != nil {
		cloudServiceSpec.CanaryDeployment = CanaryDeployment{
			Image:      continuousDeployment.CanaryDeployment.Image,
			Replicas:   continuousDeployment.CanaryDeployment.Replicas,
			Config:     continuousDeployment.CanaryDeployment.Config,
			TrafficPct: continuousDeployment.CanaryDeployment.TrafficPct,
		}
	}
	obj := NewUnstructured(CloudServiceKind)
	obj.SetLabels(utils.MergeMaps(cluster.GetLabels(), workspace.GetLabels(), project.GetLabels(), service.GetLabels()))
	obj.SetName(service.Name)
	obj.SetNamespace(service.Namespace)
	SetSpec(obj, cloudServiceSpec)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	_, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		if k8sErr.IsNotFound(err) {
			err = CreateResource(ctx, dynamicClient, obj)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		err = UpdateResource(ctx, dynamicClient, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ServiceRuntime) GetServiceStatus(ctx context.Context, service *biz.Service) error {
	obj := NewUnstructured(CloudServiceKind)
	obj.SetName(service.Name)
	obj.SetNamespace(service.Namespace)
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return err
	}
	obj, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		return err
	}
	cloudServiceStatus, found, err := unstructured.NestedInt64(obj.Object, "status")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	service.Status = biz.ServiceStatus(cloudServiceStatus)
	return nil
}
