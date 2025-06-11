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

log "Exec cluster reset..."
if ! kubeadm reset --force; then
      log "Error: Failed to reset cluster."
      exit 1
fi

rm -rf $HOME/.kube && rm -rf /etc/kubernetes && rm -rf /etc/cni

systemctl stop containerd && systemctl disable containerd && rm -rf /var/lib/containerd

systemctl stop kubelet && systemctl disable kubelet && rm -rf /var/lib/kubelet

log "Cluster reset success."
