package utils

import "fmt"

type CloudSowftwareVersion struct {
	KubernetesVersion           string   `json:"kubernetes_version"`
	KubeadmVersion              []string `json:"kubeadm_version"`
	KubectlVersion              []string `json:"kubectl_version"`
	KubeletVersion              []string `json:"kubelet_version"`
	KubernetesProxyVersion      []string `json:"kubernetes_proxy_version"`
	ContainerdVersion           []string `json:"containerd_version"`
	KubernetesCniPluginsVersion []string `json:"kubernetes_cni_plugins_version"`
	KubernetesCniVersion        []string `json:"kubernetes_cni_version"`
	CrioVersion                 []string `json:"crio_version"`
	CalicoVersion               []string `json:"calico_version"`
	FlannelVersion              []string `json:"flannel_version"`
	CoreDNSVersion              []string `json:"coredns_version"`
	CiliumVersion               []string `json:"cilium_version"`
	MetricsServerVersion        []string `json:"metrics_server_version"`
	EtcdVersion                 []string `json:"etcd_version"`
	NginxIngressVersion         []string `json:"nginx_ingress_version"`
	TraefikVersion              []string `json:"traefik_version"`
	HAProxyVersion              []string `json:"haproxy_version"`
	RookVersion                 []string `json:"rook_version"`
	CephVersion                 []string `json:"ceph_version"`
	OpenEBSVersion              []string `json:"openebs_version"`
}

func GetCloudSowftwareVersion(kubernetesVersion string) CloudSowftwareVersion {
	cloudSowftwareVersions := []CloudSowftwareVersion{
		{
			KubernetesVersion: "1.30.0",
			CrioVersion:       []string{"1.30.5"},
			KubeadmVersion:    []string{"v1.31.0"},
			KubeletVersion:    []string{"v1.31.0"},
		},
	}
	for _, version := range cloudSowftwareVersions {
		if version.KubernetesVersion == kubernetesVersion {
			return version
		}
	}
	return CloudSowftwareVersion{}
}

func (c *CloudSowftwareVersion) GetCrioLatestVersion() string {
	if len(c.CrioVersion) == 0 {
		return ""
	}
	return c.CrioVersion[len(c.CrioVersion)-1]
}

func (c *CloudSowftwareVersion) GetKubeadmLatestVersion() string {
	if len(c.KubeadmVersion) == 0 {
		return ""
	}
	return c.KubeadmVersion[len(c.KubeadmVersion)-1]
}

func (c *CloudSowftwareVersion) GetKubeletLatestVersion() string {
	if len(c.KubeletVersion) == 0 {
		return ""
	}
	return c.KubeletVersion[len(c.KubeletVersion)-1]
}

// kubelet.service
var KubeletService = `
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs/
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=/usr/bin/kubelet
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`

// 10-kubeadm.conf
var KubeadmConfig = `
# Note: This dropin only works with kubeadm and kubelet v1.11+
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
# This is a file that "kubeadm init" and "kubeadm join" generates at runtime, populating the KUBELET_KUBEADM_ARGS variable dynamically
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
# This is a file that the user can use for overrides of the kubelet args as a last resort. Preferably, the user should use
# the .NodeRegistration.KubeletExtraArgs object in the configuration files instead. KUBELET_EXTRA_ARGS should be sourced from this file.
EnvironmentFile=-/etc/sysconfig/kubelet
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
`

// kubeadm-init.conf
var KubeadmInitConfig = fmt.Sprintf(`
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: %s
  bindPort: 6443
`, "192.168.1.1")

// kubeadm-cluster.conf
var KubeadmClusterConfig = `
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
kubernetesVersion: v1.30.0
imageRepository: registry.aliyuncs.com/google_containers
controlPlaneEndpoint: "your-control-plane-endpoint:6443"
networking:
  podSubnet: "10.244.0.0/16"
`

// kubeadm-join.conf
var KubeadmJoinConfig = `
apiVersion: kubeadm.k8s.io/v1beta4
kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "node-role.kubernetes.io/master"
`

var KubeadmResetConfig = `
apiVersion: kubeadm.k8s.io/v1beta4
kind: ResetConfiguration
`

var KubeadmUpgradeConfig = `
apiVersion: kubeadm.k8s.io/v1beta4
kind: UpgradeConfiguration
`

var KubeProxyConfig = `
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
`

var KubeletConfig = `
apiVersion: kubelet.config.k8s.io/v1
kind: KubeletConfiguration
`
