package infrastructure

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
)

const (
	defaultSHHPort                = 22
	KubernetesResrouceName string = "kubernetes"
	ContainerdResrouceName string = "containerd"
	RuncResrouceName       string = "runc"
)

var (
	TimeOutPerInstance time.Duration = 5 * time.Minute

	TimeOutCountNumber               = 10 // 10 * 5s = 50s
	TimeOutSecond      time.Duration = 5  // 5s

	CloudCopilotInstallShell string = "cloud-copilot-install.sh"

	KubeadmCaTokenShell      string = "kubeadm-catoken.sh"
	KubernetesInitShell      string = "kubernetes-init.sh"
	KubernetesJoinShell      string = "kubernetes-join.sh"
	KubernetesResetShell     string = "kubernetes-reset.sh"
	KubernetesComponentShell string = "kubernetes-component.sh"

	NodeInitShell   string = "nodeinit.sh"
	SystemInfoShell string = "systeminfo.sh"

	ClusterConfiguration string = "kubernetes-config.yaml"

	ClusterController string = "controller"
	GetCaHash         string = "get-ca-hash"
	GetToken          string = "get-token"

	DefaultRootUser string = "root"
)

func getNodeArchToCloudType(arch biz.NodeArchType) string {
	switch arch {
	case biz.NodeArchType_AMD64:
		return "x86_64"
	case biz.NodeArchType_ARM64:
		return "arm64"
	default:
		return ""
	}
}

func getNodeArchByBareMetal(arch string) biz.NodeArchType {
	switch arch {
	case "x86_64":
		return biz.NodeArchType_AMD64
	case "aarch64":
		return biz.NodeArchType_ARM64
	case "arm":
		return biz.NodeArchType_ARM64
	case "arm64":
		return biz.NodeArchType_ARM64
	default:
		return biz.NodeArchType_UNSPECIFIED
	}
}

func getGPUSpecByBareMetal(gpuSpec string) biz.NodeGPUSpec {
	var GPUSpecMap = map[string]biz.NodeGPUSpec{
		"nvidia-a10":  biz.NodeGPUSpec_NVIDIA_A10,
		"nvidia-v100": biz.NodeGPUSpec_NVIDIA_V100,
		"nvidia-t4":   biz.NodeGPUSpec_NVIDIA_T4,
		"nvidia-p100": biz.NodeGPUSpec_NVIDIA_P100,
		"nvidia-p4":   biz.NodeGPUSpec_NVIDIA_P4,
	}
	if gpuSpec == "" {
		return biz.NodeGPUSpec_UNSPECIFIED
	}
	if val, ok := GPUSpecMap[gpuSpec]; ok {
		return val
	}
	return biz.NodeGPUSpec_UNSPECIFIED
}

type FindInstanceTypeParam struct {
	Os            string
	CPU           int32
	GPU           int32
	Memory        int32
	GPUSpec       biz.NodeGPUSpec
	Arch          biz.NodeArchType
	NodeGroupType biz.NodeGroupType
}

func getKubernetesVersion(resourcePath string) string {
	versionNames, err := utils.ListDirectories(filepath.Join(resourcePath, biz.NodeArchType_ARM64.String(), KubernetesResrouceName))
	if err != nil {
		return ""
	}
	if len(versionNames) > 0 {
		slices.Sort(versionNames)
		return versionNames[len(versionNames)-1]
	}
	return ""
}

func getContainerdVersion(resourcePath string) string {
	versionNames, err := utils.ListDirectories(filepath.Join(resourcePath, biz.NodeArchType_ARM64.String(), ContainerdResrouceName))
	if err != nil {
		return ""
	}
	if len(versionNames) > 0 {
		slices.Sort(versionNames)
		return versionNames[len(versionNames)-1]
	}
	return ""
}

func getRuncVersion(resourcePath string) string {
	versionNames, err := utils.ListDirectories(filepath.Join(resourcePath, biz.NodeArchType_ARM64.String(), RuncResrouceName))
	if err != nil {
		return ""
	}
	if len(versionNames) > 0 {
		slices.Sort(versionNames)
		return versionNames[len(versionNames)-1]
	}
	return ""
}

func getDefaultKuberentesImageRepo() string {
	return "registry.k8s.io"
}

func getAliyunKuberentesImageRepo() string {
	return "registry.aliyuncs.com/google_containers"
}

func getInstallShell(shellDir string, cluster *biz.Cluster) (string, error) {
	clusterJsonByte, err := json.Marshal(cluster)
	if err != nil {
		return "", err
	}
	return utils.TransferredMeaningString(map[string]string{"ClusterJsonData": string(clusterJsonByte)},
		filepath.Join(shellDir, CloudCopilotInstallShell))
}
