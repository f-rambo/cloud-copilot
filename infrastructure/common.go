package infrastructure

import (
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
)

var (
	InstallShell string = "install.sh"
)

const (
	TimeoutPerInstance = 5 * time.Minute
	TimeOutCountNumber = 10
	TimeOutSecond      = 5
)

var NodeArchToMagecloudType = map[biz.NodeArchType]string{
	biz.NodeArchType_UNSPECIFIED: "",
	biz.NodeArchType_AMD64:       "x86_64",
	biz.NodeArchType_ARM64:       "arm64",
}

var NodeArchToCloudType = map[biz.NodeArchType]string{
	biz.NodeArchType_UNSPECIFIED: "",
	biz.NodeArchType_AMD64:       "X86",
	biz.NodeArchType_ARM64:       "ARM",
}

var NodeGPUSpecToCloudSpec = map[biz.NodeGPUSpec]string{
	biz.NodeGPUSpec_UNSPECIFIED: "",
	biz.NodeGPUSpec_NVIDIA_A10:  "NVIDIA A10",
	biz.NodeGPUSpec_NVIDIA_P100: "NVIDIA P100",
	biz.NodeGPUSpec_NVIDIA_P4:   "NVIDIA P4",
	biz.NodeGPUSpec_NVIDIA_V100: "NVIDIA V100",
	biz.NodeGPUSpec_NVIDIA_T4:   "NVIDIA T4",
}

const defaultSHHPort = 22

var ARCH_MAP = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

var ArchMap = map[string]biz.NodeArchType{
	"x86_64":  biz.NodeArchType_AMD64,
	"aarch64": biz.NodeArchType_ARM64,
}

var GPUSpecMap = map[string]biz.NodeGPUSpec{
	"nvidia-a10":  biz.NodeGPUSpec_NVIDIA_A10,
	"nvidia-v100": biz.NodeGPUSpec_NVIDIA_V100,
	"nvidia-t4":   biz.NodeGPUSpec_NVIDIA_T4,
	"nvidia-p100": biz.NodeGPUSpec_NVIDIA_P100,
	"nvidia-p4":   biz.NodeGPUSpec_NVIDIA_P4,
}

var (
	Resource string = "resource"
	Shell    string = "shell"

	NodeInitShell   string = "nodeinit.sh"
	ComponentShell  string = "component.sh"
	SystemInfoShell string = "systeminfo.sh"
	ClusterInstall  string = "clusterinstall.sh"

	ClusterConfiguration string = "cluster-config.yaml"
	Install              string = "install.yaml"

	ClusterInitAction string = "init"
	ClusterJoinAction string = "join"
	ClusterController string = "controller"
)

type FindInstanceTypeParam struct {
	Os            string
	CPU           int32
	GPU           int32
	Memory        int32
	GPUSpec       biz.NodeGPUSpec
	Arch          biz.NodeArchType
	NodeGroupType biz.NodeGroupType
}

// kubernetes_version: "v1.31.2"
// containerd_version: "v2.0.0"
// runc_version: "v1.2.1"

func getKubernetesVersion() string {
	// todo find resource file version
	return "v1.31.2"
}

// func getContainerdVersion() string {
// 	// todo find resource file version
// 	return "v2.0.0"
// }

// func getRuncVersion() string {
// 	// todo find resource file version
// 	return "v1.2.1"
// }

func getDefaultKuberentesImageRepo() string {
	return "registry.k8s.io"
}

func getAliyunKuberentesImageRepo() string {
	return "registry.aliyuncs.com/google_containers"
}
