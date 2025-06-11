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

api_server=$1
caHash=$2
token=$3
is_control_plane=$4

if [ -z "$api_server" ]; then
      log "Error: API server is required."
      exit 1
fi
if [ -z "$caHash" ]; then
      log "Error: CA hash is required."
      exit 1
fi
if [ -z "$token" ]; then
      log "Error: Token is required."
      exit 1
fi

log "Exec cluster join..."

join_command="kubeadm join $api_server:6443 --token $token --discovery-token-ca-cert-hash sha256:$caHash --v=5"

if [ -n "$is_control_plane" ]; then
      log "Joining as control plane node..."
      join_command="$join_command --control-plane"
else
      log "Joining as worker node..."
fi

if ! eval "$join_command"; then
      log "Error: Failed to join cluster."
      kubeadm reset --force
      exit 1
fi

log "Cluster join success."
