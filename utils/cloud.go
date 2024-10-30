package utils

import "fmt"

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

// kubeadm.conf
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
