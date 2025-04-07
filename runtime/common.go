package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	CloudGroup       = "cloud-copilot.operator.io"
	CloudVersion     = "v1alpha1"
	DefaultNamespace = "default"

	WaitTimeout = 10 * time.Minute
)

func NewUnstructured(kindName string) *unstructured.Unstructured {
	obj := new(unstructured.Unstructured)
	obj.Object = make(map[string]any)
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CloudGroup,
		Version: CloudVersion,
		Kind:    kindName,
	})
	obj.SetNamespace(DefaultNamespace)
	return obj
}

func NewUnstructuredWithGenerateName(kindName, namePrefix string) *unstructured.Unstructured {
	obj := NewUnstructured(kindName)
	obj.SetGenerateName(namePrefix + "-")
	return obj
}

func SetSpec(obj *unstructured.Unstructured, spec any) {
	obj.Object["spec"] = spec
}

// SetSpecField(obj, "template.metadata.labels.app", "myapp")
func SetSpecField(obj *unstructured.Unstructured, fieldPath string, value any) error {
	return unstructured.SetNestedField(obj.Object, value, "spec", fieldPath)
}

// Convert the spec of an unstructured object to a specified struct
func GetSpec(obj *unstructured.Unstructured, out any) error {
	if obj == nil || out == nil {
		return errors.New("input object cannot be nil")
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return err
	}
	if !found {
		return errors.New("spec not found")
	}
	spceYamlByte, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(spceYamlByte, out)
	if err != nil {
		return err
	}
	return nil
}

func GetStatus(obj *unstructured.Unstructured) (status any, err error) {
	if obj == nil {
		return nil, errors.New("input object cannot be nil")
	}
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("status not found")
	}
	return status, nil
}

// err = c.waitForCRDStatus(ctx, dynamicClient, namespace, name, gvr, 5*time.Minute, func(obj *unstructured.Unstructured) (bool, error) {
// 	status, found, err := unstructured.NestedString(obj.Object, "status", "phase")
// 	if err != nil || !found {
// 	    return false, err
// 	}
// 	return status == "Ready", nil
//   })

func WaitForCRDStatus(ctx context.Context, dynamicClient *dynamic.DynamicClient, res *unstructured.Unstructured, timeout time.Duration, condition func(obj *unstructured.Unstructured) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := dynamicClient.Resource(res.GroupVersionKind().GroupVersion().WithResource(res.GetKind())).
			Namespace(res.GetNamespace()).Get(ctx, res.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return condition(obj)
	})
}

func CreateYAMLFile(ctx context.Context, dynamicClient *dynamic.DynamicClient, namespace, resource, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "open file failed")
	}
	defer file.Close()
	decoder := k8syaml.NewYAMLOrJSONDecoder(file, 1024)
	for {
		unstructuredObj := &unstructured.Unstructured{}
		if getErr := decoder.Decode(unstructuredObj); err != nil {
			if getErr.Error() == "EOF" {
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

func ParseYAML(filename string) (*unstructured.UnstructuredList, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	list := &unstructured.UnstructuredList{Items: make([]unstructured.Unstructured, 0)}
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var obj map[string]any
		err = decoder.Decode(&obj)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		list.Items = append(list.Items, unstructured.Unstructured{Object: obj})
	}
	return list, nil
}

func GetKubeDynamicClient(KubeConfigPaths ...string) (*dynamic.DynamicClient, error) {
	var KubeConfigPath string
	if len(KubeConfigPaths) == 0 {
		KubeConfigPath = clientcmd.RecommendedHomeFile
	} else {
		KubeConfigPath = KubeConfigPaths[0]
	}
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes by kubeconfig client failed")
	}
	return client, nil
}

func GetKubeClient(KubeConfigPaths ...string) (clientset *kubernetes.Clientset, err error) {
	var KubeConfigPath string
	if len(KubeConfigPaths) == 0 {
		KubeConfigPath = clientcmd.RecommendedHomeFile
	} else {
		KubeConfigPath = KubeConfigPaths[0]
	}
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes by kubeconfig client failed")
	}
	return client, nil
}

func CreateResource(ctx context.Context, client dynamic.Interface, resource *unstructured.Unstructured) error {
	gvk := resource.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}
	_, err := client.Resource(gvr).Namespace(resource.GetNamespace()).Create(ctx, resource, metav1.CreateOptions{})
	return err
}

func DeleteResource(ctx context.Context, client dynamic.Interface, resource *unstructured.Unstructured) error {
	gvk := resource.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}
	err := client.Resource(gvr).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), metav1.DeleteOptions{})
	return err
}

func GetResource(ctx context.Context, client dynamic.Interface, resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvk := resource.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}
	return client.Resource(gvr).Namespace(resource.GetNamespace()).Get(ctx, resource.GetName(), metav1.GetOptions{})
}

func UpdateResource(ctx context.Context, client dynamic.Interface, resource *unstructured.Unstructured) error {
	gvk := resource.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}
	_, err := client.Resource(gvr).Namespace(resource.GetNamespace()).Update(ctx, resource, metav1.UpdateOptions{})
	return err
}

func WaitForPodReady(ctx context.Context, client *kubernetes.Clientset, namespace, podName string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second*2, timeout, true, func(ctx context.Context) (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" || pod.Status.Phase == "Completed" {
			return true, nil
		}

		if pod.Status.Phase != "Running" {
			return false, nil
		}

		readyContainers := 0
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Ready {
				readyContainers++
			}
		}

		totalContainers := len(pod.Spec.Containers)
		if readyContainers != totalContainers {
			return false, nil
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" {
				return condition.Status == "True", nil
			}
		}

		return false, nil
	})
}

func GetObjResrouce(ctx context.Context, obj *unstructured.Unstructured, successStatus, failedStatus int32) (*unstructured.Unstructured, error) {
	dynamicClient, err := GetKubeDynamicClient()
	if err != nil {
		return nil, err
	}
	err = CreateResource(ctx, dynamicClient, obj)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = DeleteResource(ctx, dynamicClient, obj)
	}()
	err = WaitForCRDStatus(ctx, dynamicClient, obj, WaitTimeout, func(obj *unstructured.Unstructured) (bool, error) {
		status, found, getErr := unstructured.NestedMap(obj.Object, "status")
		if getErr != nil {
			return false, getErr
		}
		if !found {
			return false, nil
		}
		statusValue, ok := status["status"]
		if !ok {
			return false, nil
		}
		if cast.ToInt32(statusValue) == failedStatus {
			return false, errors.New("status is failed")
		}
		return cast.ToInt32(statusValue) == successStatus, nil
	})
	if err != nil {
		return nil, err
	}
	obj, err = GetResource(ctx, dynamicClient, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// kubectl describe resource -n namespace
func DescribeResource(ctx context.Context, client *kubernetes.Clientset, namespace, resourceName string) (string, error) {
	return "", nil
}

// kubectl logs -n namespace podName
func Logs(ctx context.Context, client *kubernetes.Clientset, namespace, podName string) (string, error) {
	return "", nil
}

// kubectl describe resource -n namespace
func PodInfo(ctx context.Context, client *kubernetes.Clientset, namespace, podName string) (string, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name:\t%s\n", pod.Name))
	result.WriteString(fmt.Sprintf("Namespace:\t%s\n", pod.Namespace))
	result.WriteString(fmt.Sprintf("Node:\t%s\n", pod.Spec.NodeName))
	result.WriteString(fmt.Sprintf("Status:\t%s\n", pod.Status.Phase))
	result.WriteString(fmt.Sprintf("IP:\t%s\n", pod.Status.PodIP))

	result.WriteString("\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		result.WriteString(fmt.Sprintf("  - Name:\t%s\n", container.Name))
		result.WriteString(fmt.Sprintf("    Image:\t%s\n", container.Image))
		result.WriteString(fmt.Sprintf("    Ports:\t%v\n", container.Ports))
	}

	return result.String(), nil
}

// kubectl logs -n namespace podName
func PodLogs(ctx context.Context, client *kubernetes.Clientset, namespace, podName, containerName string) (string, error) {
	podLogOpts := &corev1.PodLogOptions{
		Container: containerName, // If Container is empty, the first container in the pod will be chosen
		Follow:    false,
		Previous:  false,
		TailLines: utils.Int64Ptr(100),
	}
	req := client.CoreV1().Pods(namespace).GetLogs(podName, podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func EventsInfo(ctx context.Context, client *kubernetes.Clientset, namespace, kind, name string) (string, error) {
	var result strings.Builder
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=%s",
			name, namespace, kind),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get events: %v", err)
	}

	if len(events.Items) > 0 {
		result.WriteString("\nEvents:\n")
		result.WriteString("Type\tReason\tAge\tFrom\tMessage\n")
		result.WriteString("----\t------\t---\t----\t-------\n")

		for _, event := range events.Items {
			age := time.Since(event.FirstTimestamp.Time).Round(time.Second)
			result.WriteString(fmt.Sprintf("%v\t%v\t%v\t%v\t%v\n",
				event.Type,
				event.Reason,
				age,
				event.Source.Component,
				event.Message,
			))
		}
	}
	return result.String(), nil
}

// ResourceInfo
func ResourceInfo(ctx context.Context, client dynamic.Interface, namespace, name string, gvr schema.GroupVersionResource) (string, error) {
	resource, err := client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var result strings.Builder

	result.WriteString(fmt.Sprintf("Name:\t%s\n", resource.GetName()))
	result.WriteString(fmt.Sprintf("Namespace:\t%s\n", resource.GetNamespace()))
	result.WriteString(fmt.Sprintf("Kind:\t%s\n", resource.GetKind()))
	result.WriteString(fmt.Sprintf("API Version:\t%s\n", resource.GetAPIVersion()))
	result.WriteString(fmt.Sprintf("Creation Timestamp:\t%s\n", resource.GetCreationTimestamp()))

	if labels := resource.GetLabels(); len(labels) > 0 {
		result.WriteString("\nLabels:\n")
		for k, v := range labels {
			result.WriteString(fmt.Sprintf("  %s:\t%s\n", k, v))
		}
	}

	if annotations := resource.GetAnnotations(); len(annotations) > 0 {
		result.WriteString("\nAnnotations:\n")
		for k, v := range annotations {
			result.WriteString(fmt.Sprintf("  %s:\t%s\n", k, v))
		}
	}

	if spec, ok := resource.Object["spec"]; ok {
		result.WriteString("\nSpec:\n")
		formatUnstructuredField(&result, spec, 2)
	}

	if status, ok := resource.Object["status"]; ok {
		result.WriteString("\nStatus:\n")
		formatUnstructuredField(&result, status, 2)
	}
	return result.String(), nil
}

func formatUnstructuredField(builder *strings.Builder, field any, indent int) {
	indentStr := strings.Repeat("  ", indent)

	switch v := field.(type) {
	case map[string]any:
		for key, value := range v {
			builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, key))
			formatUnstructuredField(builder, value, indent+1)
		}
	case []any:
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s- ", indentStr))
			formatUnstructuredField(builder, item, indent+1)
		}
	default:
		builder.WriteString(fmt.Sprintf("%s%v\n", indentStr, v))
	}
}
