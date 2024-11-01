#!/bin/bash
set -e

if [ -z "${1// /}" ] || [ -z "${2// /}" ]; then
  echo "Usage: $0 <RESOURCE> <CLUSTER_VERSION>"
  exit 1
fi

KUBERNETES_VERSION=${1:-"v1.31.2"}
CONTAINERD_VERSION=${2:-"v1.7.23"}
RUNC_VERSION=${3:-"v1.2.0"}
CNIPLUGINS_VERSION=${4:-"v1.6.0"}
RESOURCE=${5:-"$HOME/resource"}

if [[ ! $(realpath "$RESOURCE") =~ ^/ ]]; then
  echo "Error: RESOURCE must be an absolute path"
  exit 1
fi

if [[ ! $CLUSTER_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
  echo "Error: CLUSTER_VERSION must be in the format X.Y.Z or X.Y.Z-<suffix>"
  exit 1
fi

if [ ! -d "$RESOURCE" ] || [ ! -r "$RESOURCE" ]; then
  echo "Error: RESOURCE directory $RESOURCE does not exist or is not readable"
  exit 1
fi

if [ ! -d "$RESOURCE/kubernetes-software/$CLUSTER_VERSION" ]; then
  echo "Error: CLUSTER_VERSION $CLUSTER_VERSION does not exist in the RESOURCE directory"
  exit 1
fi

ARCH=$(uname -m)
case $ARCH in
aarch64)
  ARCH="arm64"
  ;;
x86_64)
  ARCH="amd64"
  ;;
*)
  echo "Error: Unsupported architecture $ARCH. Supported architectures are: aarch64, x86_64"
  exit 1
  ;;
esac

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" != "linux" ]]; then
  echo "Error: Unsupported OS $OS"
  exit 1
fi

containerdService=$(
  cat <<EOF
[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target local-fs.target dbus.service

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/containerd

Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5

LimitNPROC=infinity
LimitCORE=infinity

# Comment TasksMax if your systemd version does not supports it.
# Only systemd 226 and above support this version.
TasksMax=infinity
OOMScoreAdjust=-999

[Install]
WantedBy=multi-user.target
EOF
)

function install_containerd() {
  echo "install containerd..."
  containerdPath="$RESOURCE/$ARCH/containerd/$CONTAINERD_VERSION/"
  if [ ! -d "$containerdPath" ] || [ ! -r "$containerdPath" ]; then
    echo "Error: Directory $containerdPath does not exist or is not readable"
    exit 1
  fi
  sudo mv "$containerdPath/bin/*" /usr/local/bin/
  if ! ctr --version; then
    echo "Error: Failed to start containerd service"
    exit 1
  fi
  ctr config default | sed "s/SystemdCgroup: false/SystemdCgroup: true/g" | sudo tee /etc/containerd/config.toml

  if ! echo "$containerdService" | sudo tee /usr/lib/systemd/system/containerd.service >/dev/null; then
    echo "Error: Failed to write to /usr/lib/systemd/system/containerd.service"
    exit 1
  fi
  if ! sudo systemctl daemon-reload || ! sudo systemctl enable --now containerd; then
    echo "Error: Failed to start containerd service"
    exit 1
  fi

  echo "install runc..."
  runcPath="$RESOURCE/$ARCH/runc/$RUNC_VERSION/"
  if [ ! -d "$runcPath" ] || [ ! -r "$runcPath" ]; then
    echo "Error: Directory $runcPath does not exist or is not readable"
    exit 1
  fi
  sudo install -m 755 "$runcPath/runc" /usr/local/bin/runc

  echo "install cni plugins..."
  sudo mkdir -p /opt/cni/bin
  cnipluginsPath="$RESOURCE/$ARCH/cni-plugins/$CNIPLUGINS_VERSION/"
  if [ ! -d "$cnipluginsPath" ] || [ ! -r "$cnipluginsPath" ]; then
    echo "Error: Directory $cnipluginsPath does not exist or is not readable"
    exit 1
  fi
  sudo mv "$cnipluginsPath/*" /opt/cni/bin/
}

kubeadmConfig=$(
  cat <<EOF
# Note: This dropin only works with kubeadm and kubelet v1.11+
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
# This is a file that "kubeadm init" and "kubeadm join" generates at runtime, populating the KUBELET_KUBEADM_ARGS variable dynamically
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
# This is a file that the user can use for overrides of the kubelet args as a last resort. Preferably, the user should use
# the .NodeRegistration.KubeletExtraArgs object in the configuration files instead. KUBELET_EXTRA_ARGS should be sourced from this file.
EnvironmentFile=-/etc/sysconfig/kubelet
ExecStart=/usr/local/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
EOF
)

kubeletService=$(
  cat <<EOF
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs/
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=/usr/local/bin/kubelet
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
)

function install_kubernetes_software() {
  kubernetesPath="$RESOURCE/$ARCH/kubernetes/$KUBERNETES_VERSION/"

  if [ ! -d "$kubernetesPath" ] || [ ! -r "$kubernetesPath" ]; then
    echo "Error: Directory $kubernetesPath does not exist or is not readable"
    exit 1
  fi

  if [ ! -f "$kubernetesPath/kubeadm" ]; then
    echo "Error: File $kubernetesPath/kubeadm does not exist"
    exit 1
  fi

  if [ ! -x "$kubernetesPath/kubeadm" ]; then
    sudo chmod +x "$kubernetesPath/kubeadm"
    sudo mv "$kubernetesPath/kubeadm" /usr/local/bin/kubeadm
  fi

  if [ ! -f "$kubernetesPath/kubectl" ]; then
    echo "Error: File $kubernetesPath/kubectl does not exist"
    exit 1
  fi

  if [ ! -x "$kubernetesPath/kubectl" ]; then
    sudo chmod +x "$kubernetesPath/kubectl"
    sudo mv "$kubernetesPath/kubectl" /usr/local/bin/kubectl
  fi

  if ! echo "$kubeadmConfig" | sed "s:/usr/bin:/usr/local/bin:g" | sudo tee /usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf >/dev/null; then
    echo "Error: Failed to write to /usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf"
    exit 1
  fi

  if ! echo "$kubeletService" | sed "s:/usr/bin:/usr/local/bin:g" | sudo tee /usr/lib/systemd/system/kubelet.service >/dev/null; then
    echo "Error: Failed to write to /usr/lib/systemd/system/kubelet.service"
    exit 1
  fi

  if ! sudo systemctl daemon-reload || ! sudo systemctl enable --now kubelet; then
    echo "Error: Failed to start kubelet service"
    exit 1
  fi
}

function install_kubernetes_images() {
  kubernetes_images_path="$RESOURCE/${ARCH}/kubernetes/${KUBERNETES_VERSION}/kubernetes-images.tar"
  if [ ! -f "$kubernetes_images_path" ]; then
    echo "Error: File $kubernetes_images_path does not exist"
    exit 1
  fi

  if systemctl is-active --quiet containerd; then
    sudo ctr -n=k8s.io images import "$kubernetes_images_path"
  fi
}

if systemctl is-active --quiet containerd; then
  echo "containerd is already running, skipping installation."
else
  echo "containerd is not running, proceeding with installation."
  install_containerd
fi

if systemctl is-active --quiet kubelet; then
  echo "kubelet is already running, skipping installation."
else
  echo "kubelet is not running, proceeding with installation."
  install_kubernetes_software
fi

install_kubernetes_images
