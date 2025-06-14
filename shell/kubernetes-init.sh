#!/bin/bash
set -e

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

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

if [ -n "$SUDO_USER" ]; then
      ORIGINAL_USER=$SUDO_USER
      ORIGINAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
      ORIGINAL_USER=$USER
      ORIGINAL_HOME=$HOME
fi

KUBERNETES_VERSION=${1:-"v1.31.2"}

if [ -z "$KUBERNETES_VERSION" ]; then
      log "Error: Kubernetes version is required."
      exit 1
fi

log "Exec cluster init..."

cluster_config_path=$ORIGINAL_HOME/resource/kubernetes-config.yaml
if [ ! -f $cluster_config_path ]; then
      log "Error: Cluster config file not found."
      exit 1
fi

if ! kubeadm init --config $cluster_config_path --v=5; then
      log "Error: Failed to init cluster."
      kubeadm reset --force
      exit 1
fi

log "Cluster init success."

rm -f $ORIGINAL_HOME/.kube/config && mkdir -p $ORIGINAL_HOME/.kube && sudo cp -i /etc/kubernetes/admin.conf $ORIGINAL_HOME/.kube/config && chown $ORIGINAL_USER:$ORIGINAL_USER $ORIGINAL_HOME/.kube/config
