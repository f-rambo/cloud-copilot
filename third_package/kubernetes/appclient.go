package kubernetes

import (
	"context"
	"fmt"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppDeployedRuntime struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewAppDeployedResource(c *conf.Bootstrap, logger log.Logger) biz.AppRuntime {
	return &AppDeployedRuntime{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (a *AppDeployedRuntime) GetPodResources(ctx context.Context, appDeployed *biz.DeployApp) ([]*biz.AppDeployedResource, error) {
	resources := make([]*biz.AppDeployedResource, 0)
	clusterClient, err := getKubeClient("")
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	podResources, _ := clusterClient.CoreV1().Pods(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if podResources != nil && len(podResources.Items) > 0 {
		for _, pod := range podResources.Items {
			resource := &biz.AppDeployedResource{
				Name:      pod.Name,
				Kind:      "Pod",
				StartedAt: pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status:    []string{string(pod.Status.Phase)},
				Events:    make([]string, 0),
			}
			events, _ := clusterClient.CoreV1().Events(appDeployed.Namespace).List(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			})
			if events != nil && len(events.Items) > 0 {
				for _, event := range events.Items {
					resource.Events = append(resource.Events, event.Message)
				}
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (a *AppDeployedRuntime) GetNetResouces(ctx context.Context, appDeployed *biz.DeployApp) ([]*biz.AppDeployedResource, error) {
	resources := make([]*biz.AppDeployedResource, 0)
	clusterClient, err := getKubeClient("")
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	serviceResources, _ := clusterClient.CoreV1().Services(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if serviceResources != nil && len(serviceResources.Items) > 0 {
		for _, service := range serviceResources.Items {
			port := ""
			for _, v := range service.Spec.Ports {
				port = fmt.Sprintf("%s %d:%d/%s", port, v.Port, v.NodePort, v.Protocol)
			}
			externalIPs := ""
			for _, v := range service.Spec.ExternalIPs {
				externalIPs = fmt.Sprintf("%s,%s", externalIPs, v)
			}
			resource := &biz.AppDeployedResource{
				Name:      service.Name,
				Kind:      "Service",
				StartedAt: service.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Type: " + string(service.Spec.Type),
					"ClusterIP: " + service.Spec.ClusterIP,
					"ExternalIP: " + externalIPs,
					"Port: " + port,
				},
			}
			resources = append(resources, resource)
		}
	}
	ingressResources, _ := clusterClient.NetworkingV1beta1().Ingresses(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if ingressResources != nil && len(ingressResources.Items) > 0 {
		for _, ingress := range ingressResources.Items {
			class := ""
			if ingress.Spec.IngressClassName != nil {
				class = *ingress.Spec.IngressClassName
			}
			hosts := ""
			for _, v := range ingress.Spec.Rules {
				hosts = fmt.Sprintf("%s,%s", hosts, v.Host)
			}
			ports := ""
			for _, v := range ingress.Spec.TLS {
				for _, v := range v.Hosts {
					ports = fmt.Sprintf("%s,%s", ports, v)
				}
			}
			loadBalancerIP := ""
			for _, v := range ingress.Status.LoadBalancer.Ingress {
				loadBalancerIP = fmt.Sprintf("%s,%s", loadBalancerIP, v.IP)
			}
			resource := &biz.AppDeployedResource{
				Name:      ingress.Name,
				Kind:      "Ingress",
				StartedAt: ingress.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"class: " + class,
					"hosts: " + hosts,
					"address: " + loadBalancerIP,
					"ports: " + ports,
				},
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (a *AppDeployedRuntime) GetAppsReouces(ctx context.Context, appDeployed *biz.DeployApp) ([]*biz.AppDeployedResource, error) {
	resources := make([]*biz.AppDeployedResource, 0)
	clusterClient, err := getKubeClient("")
	if err != nil {
		return nil, err
	}
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", appDeployed.ReleaseName)
	deploymentResources, _ := clusterClient.AppsV1().Deployments(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if deploymentResources != nil && len(deploymentResources.Items) > 0 {
		for _, deployment := range deploymentResources.Items {
			resource := &biz.AppDeployedResource{
				Name:      deployment.Name,
				Kind:      "Deployment",
				StartedAt: deployment.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Ready: " + cast.ToString(deployment.Status.ReadyReplicas),
					"Up-to-date: " + cast.ToString(deployment.Status.UpdatedReplicas),
					"Available: " + cast.ToString(deployment.Status.AvailableReplicas),
				},
			}
			resources = append(resources, resource)
		}
	}

	statefulSetResources, _ := clusterClient.AppsV1().StatefulSets(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if statefulSetResources != nil && len(statefulSetResources.Items) > 0 {
		for _, statefulSet := range statefulSetResources.Items {
			resource := &biz.AppDeployedResource{
				Name:      statefulSet.Name,
				Kind:      "StatefulSet",
				StartedAt: statefulSet.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Ready: " + cast.ToString(statefulSet.Status.ReadyReplicas),
					"Up-to-date: " + cast.ToString(statefulSet.Status.UpdatedReplicas),
					"Available: " + cast.ToString(statefulSet.Status.AvailableReplicas),
				},
			}
			resources = append(resources, resource)
		}
	}

	deamonsetResources, _ := clusterClient.AppsV1().DaemonSets(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if deamonsetResources != nil && len(deamonsetResources.Items) > 0 {
		for _, deamonset := range deamonsetResources.Items {
			resource := &biz.AppDeployedResource{
				Name:      deamonset.Name,
				Kind:      "Deamonset",
				StartedAt: deamonset.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Desired: " + cast.ToString(deamonset.Status.DesiredNumberScheduled),
					"Current: " + cast.ToString(deamonset.Status.CurrentNumberScheduled),
					"Ready: " + cast.ToString(deamonset.Status.NumberReady),
					"Up-to-date: " + cast.ToString(deamonset.Status.UpdatedNumberScheduled),
					"Available: " + cast.ToString(deamonset.Status.NumberAvailable),
				},
			}
			resources = append(resources, resource)
		}
	}

	jobResources, _ := clusterClient.BatchV1().Jobs(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if jobResources != nil && len(jobResources.Items) > 0 {
		for _, job := range jobResources.Items {
			resource := &biz.AppDeployedResource{
				Name:      job.Name,
				Kind:      "Job",
				StartedAt: job.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Completions: " + cast.ToString(job.Spec.Completions),
					"Parallelism: " + cast.ToString(job.Spec.Parallelism),
					"BackoffLimit: " + cast.ToString(job.Spec.BackoffLimit),
				},
			}
			resources = append(resources, resource)
		}
	}

	cronjobResources, _ := clusterClient.BatchV1beta1().CronJobs(appDeployed.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if cronjobResources != nil && len(cronjobResources.Items) > 0 {
		for _, cronjob := range cronjobResources.Items {
			suspend := *cronjob.Spec.Suspend // Dereference the pointer to bool
			resource := &biz.AppDeployedResource{
				Name:      cronjob.Name,
				Kind:      "Cronjob",
				StartedAt: cronjob.CreationTimestamp.Format("2006-01-02 15:04:05"),
				Status: []string{
					"Schedule: " + cronjob.Spec.Schedule,
					"Suspend: " + cast.ToString(suspend),
					"Active: " + cast.ToString(len(cronjob.Status.Active)),
					"Last Schedule: " + cronjob.Status.LastScheduleTime.Format("2006-01-02 15:04:05"),
				},
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}
