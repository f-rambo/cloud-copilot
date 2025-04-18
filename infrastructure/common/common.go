package common

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

type FindInstanceTypeParam struct {
	Os            string
	CPU           int32
	GPU           int32
	Memory        int32
	GPUSpec       biz.NodeGPUSpec
	Arch          biz.NodeArchType
	NodeGroupType biz.NodeGroupType
}
