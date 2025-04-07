#!/bin/bash
set -e

log() {
  local message="$1"
  echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

parse_yaml() {
  local yaml_string="$1"
  local query_path="$2"
  echo "$yaml_string" | yq eval "$query_path" -
}

cluster_data=$1

RESOURCE=$(parse_yaml "$cluster_data" '.resrouce_path')
IMAGE_REPO=$(parse_yaml "$cluster_data" '.image_repo')
KUBERNETES_VERSION=$(parse_yaml "$cluster_data" '.kuberentes_version')
CONTAINERD_VERSION=$(parse_yaml "$cluster_data" '.containerd_version')
RUNC_VERSION=$(parse_yaml "$cluster_data" '.runc_version')

if [ ! -d "$RESOURCE" ] || [ ! -r "$RESOURCE" ]; then
  log "Error: RESOURCE directory $RESOURCE does not exist or is not readable"
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
  log "Error: Unsupported architecture $ARCH. Supported architectures are: aarch64, x86_64"
  exit 1
  ;;
esac

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" != "linux" ]]; then
  log "Error: Unsupported OS $OS"
  exit 1
fi

function install_kubernetes_software() {
  kubernetesPath="$RESOURCE/$ARCH/kubernetes/$KUBERNETES_VERSION"

  if ! install -m 755 "$kubernetesPath/kubeadm" /usr/local/bin/kubeadm; then
    log "Error: Failed to install kubeadm"
    exit 1
  fi

  if ! install -m 755 "$kubernetesPath/kubelet" /usr/local/bin/kubelet; then
    log "Error: Failed to install kubelet"
    exit 1
  fi

  if ! cp "$kubernetesPath/kubelet.service" /usr/lib/systemd/system/; then
    log "Error: Failed to copy kubelet.service to /usr/lib/systemd/system/"
    exit 1
  fi

  if ! systemctl daemon-reload; then
    log "Error: Failed to reload systemd daemon"
    exit 1
  fi

  if ! systemctl enable kubelet; then
    log "Error: Failed to enable kubelet service"
    exit 1
  fi

}

function install_containerd() {
  log "install runc..."

  runcPath="$RESOURCE/$ARCH/runc/$RUNC_VERSION"

  if [ ! -d "$runcPath" ] || [ ! -r "$runcPath" ]; then
    log "Error: Directory $runcPath does not exist or is not readable"
    exit 1
  fi

  if ! install -C -m 755 "$runcPath/runc" /usr/local/bin/runc; then
    log "Error: Failed to install runc"
    exit 1
  fi

  log "install containerd..."

  containerdPath="$RESOURCE/$ARCH/containerd/$CONTAINERD_VERSION"

  if [ ! -d "$containerdPath" ] || [ ! -r "$containerdPath" ]; then
    log "Error: Directory $containerdPath does not exist or is not readable"
    exit 1
  fi

  if ! install -C -m 755 "$containerdPath/bin/containerd" /usr/local/bin/containerd; then
    log "Error: Failed to install containerd"
    exit 1
  fi

  if ! install -C -m 755 "$containerdPath/bin/containerd-shim-runc-v2" /usr/local/bin/containerd-shim-runc-v2; then
    log "Error: Failed to install containerd-shim-runc-v2"
    exit 1
  fi

  if ! install -C -m 755 "$containerdPath/bin/ctr" /usr/local/bin/ctr; then
    log "Error: Failed to install ctr"
    exit 1
  fi

  if ! install -C -m 755 "$containerdPath/bin/containerd-stress" /usr/local/bin/containerd-stress; then
    log "Error: Failed to install containerd-stress"
    exit 1
  fi

  mkdir -p /etc/containerd && touch /etc/containerd/config.toml

  containerd config default | sed -e '/containerd.runtimes.runc.options/a\            SystemdCgroup = true' | tee /etc/containerd/config.toml >/dev/null

  current_sandbox=$(grep "sandbox =" /etc/containerd/config.toml | awk -F"'" '{print $2}')

  image_and_tag=$(echo "$current_sandbox" | awk -F'/' '{print $NF}')

  sed -i "s|sandbox = .*|sandbox = \"${IMAGE_REPO}/${image_and_tag}\"|" /etc/containerd/config.toml

  if ! cp "$containerdPath/containerd.service" /usr/lib/systemd/system/; then
    log "Error: Copy containerd.service to /usr/lib/systemd/system/ failed"
    exit 1
  fi

  if ! systemctl daemon-reload; then
    log "Error: Failed to reload systemd daemon"
    exit 1
  fi

  if ! systemctl enable --now containerd; then
    log "Error: Failed to start containerd service"
    exit 1
  fi

  if ! systemctl restart containerd; then
    log "Error: Failed to restart containerd service"
    exit 1
  fi

  # sleep 10

  # if ! ctr namespace list | grep -q "k8s.io"; then
  #   ctr namespace create k8s.io
  # fi

  # sleep 10

  # kubernetes_image_path="$RESOURCE/$ARCH/kubernetes/$KUBERNETES_VERSION/kubernetes-images.tar"
  # if [ ! -f "$kubernetes_image_path" ] || [ ! -r "$kubernetes_image_path" ]; then
  #   log "Error: File $kubernetes_image_path does not exist or is not readable"
  #   exit 1
  # fi

  # ctr -n k8s.io images import "$kubernetes_image_path"
}

install_containerd

install_kubernetes_software

# todo install HAProxy + Keepalived load balance

log "kubernetes software and containerd installation completed successfully."

exit 0
