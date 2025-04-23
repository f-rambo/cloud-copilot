#!/bin/bash
set -e

if [ -n "$SUDO_USER" ]; then
      ORIGINAL_USER=$SUDO_USER
      ORIGINAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
      ORIGINAL_USER=$USER
      ORIGINAL_HOME=$HOME
fi

RESOURCE=${1:-"$ORIGINAL_HOME/resource"}
KUBERNETES_VERSION=${2:-"v1.31.2"}
CONTAINERD_VERSION=${3:-"v2.0.0"}
RUNC_VERSION=${4:-"v1.2.1"}
SERVICE_VERSION=${5:-"v0.0.1"}

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

ARCH=$(uname -m)
case $ARCH in
aarch64)
      ARCH="arm64"
      ;;
arm64)
      ARCH="arm64"
      ;;
x86_64)
      ARCH="amd64"
      ;;
*)
      log "Error: Unsupported architecture $ARCH"
      exit 1
      ;;
esac

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" != "linux" ]]; then
      log "Error: Unsupported OS $OS"
      exit 1
fi

create_directory() {
      local dir=$1
      if [ ! -d "$dir" ]; then
            mkdir -p "$dir" || {
                  log "Failed to create directory: $dir"
                  exit 1
            }
      fi
}

download_file() {
      local url=$1
      local file=$2
      local checksum_file=$3

      log "Downloading $url"

      if [[ -z "$url" || -z "$file" ]]; then
            log "Error: URL and file parameters are required."
            exit 1
      fi

      if [ ! -f "$file" ]; then
            if ! curl -L -C - --fail -O "$url"; then
                  log "Failed to download $url"
                  rm -f "$file"
                  exit 1
            fi

            if [ -n "$checksum_file" ]; then
                  if ! curl -L -C - --fail -O "$url.sha256sum"; then
                        log "Failed to download $url.sha256sum"
                        rm -f "$file"
                        exit 1
                  fi

            fi
      fi
}

verify_checksum() {
      local file=$1
      local checksum_file=$2

      if [[ ! -f "$file" ]]; then
            log "File not found: $file" >&2
            return 1
      fi

      sleep 10

      if [[ ! -f "$checksum_file" ]]; then
            log "Checksum file not found: $checksum_file" >&2
            return 1
      fi

      sleep 10

      if ! sha256sum -c "$checksum_file"; then
            log "SHA256 checksum verification failed for $file" >&2
            return 1
      fi

      return 0
}

extract_tar() {
      local tarfile=$1
      local dest_dir=$2
      tar -xzf "$tarfile" -C "$dest_dir" || {
            log "Failed to extract $tarfile"
            exit 1
      }
}

function download_containerd() {
      log "download containerd ${CONTAINERD_VERSION} ${ARCH}"
      containerd_path="${RESOURCE}/containerd/${CONTAINERD_VERSION}"
      create_directory "$containerd_path"
      containerd_version_num=$(echo "$CONTAINERD_VERSION" | sed 's/^v//')
      containerd_tarfile="containerd-${containerd_version_num}-linux-${ARCH}.tar.gz"
      if ! download_file "https://github.com/containerd/containerd/releases/download/${CONTAINERD_VERSION}/${containerd_tarfile}" "${containerd_tarfile}" "${containerd_tarfile}.sha256sum"; then
            log "Failed to download containerd"
            return 1
      fi
      if ! verify_checksum "$containerd_tarfile" "${containerd_tarfile}.sha256sum"; then
            log "Checksum verification failed"
            rm -f "$containerd_tarfile" "${containerd_tarfile}.sha256sum"
            return 1
      fi
      extract_tar "$containerd_tarfile" "$containerd_path"
      rm -f "$containerd_tarfile" "${containerd_tarfile}.sha256sum"

      log "download runc ${RUNC_VERSION} ${ARCH}"
      runc_path="${RESOURCE}/runc/${RUNC_VERSION}"
      create_directory "$runc_path"
      if ! download_file "https://github.com/opencontainers/runc/releases/download/${RUNC_VERSION}/runc.${ARCH}" "runc.${ARCH}"; then
            log "Failed to download runc"
            return 1
      fi
      mv runc.${ARCH} "$runc_path/runc"
}

function download_kubeadm_kubelet() {
      log "download kubeadm kubelet ${KUBERNETES_VERSION} ${ARCH}"
      kubernetes_path="${RESOURCE}/kubernetes/${KUBERNETES_VERSION}"
      create_directory "$kubernetes_path"
      if ! download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${ARCH}/kubeadm" "kubeadm"; then
            log "Failed to download kubeadm"
            return 1
      fi
      if ! download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${ARCH}/kubelet" "kubelet"; then
            log "Failed to download kubelet"
            return 1
      fi
      mv kubeadm "$kubernetes_path/kubeadm"
      mv kubelet "$kubernetes_path/kubelet"
}

function pull_images() {
      log "Pulling images..."
      local kubeadm_path="${RESOURCE}/kubernetes/${KUBERNETES_VERSION}/kubeadm"
      if [ ! -f "$kubeadm_path" ]; then
            echo "Error: kubeadm not found"
            return 1
      fi

      if ! chmod +x "$kubeadm_path"; then
            echo "Error: Failed to change permissions of $kubeadm_path"
            return 1
      fi

      local kube_images=$("$kubeadm_path" config images list --kubernetes-version "$KUBERNETES_VERSION")
      if [ $? -ne 0 ]; then
            echo "Error: Failed to get Kubernetes images list"
            return 1
      fi

      images_array=($(echo "$kube_images" | tr '\n' ' '))

      local images_dir="${RESOURCE}/kubernetes/${KUBERNETES_VERSION}/"
      if ! create_directory "$images_dir"; then
            echo "Error: Failed to create directory $images_dir"
            return 1
      fi
      local images_tarfile="${images_dir}/kubernetes-images.tar"

      # docker save calico/typha:v3.29.0 calico/kube-controllers:v3.29.0 calico/apiserver:v3.29.0 calico/csi:v3.29.0 calico/node:v3.29.0 calico/pod2daemon-flexvol:v3.29.0 calico/cni:v3.29.0 calico/node-driver-registrar:v3.29.0 -o calico.tar
      for image in "${images_array[@]}"; do
            if ! docker pull --platform=linux/$ARCH "$image"; then
                  echo "Error: Failed to pull image $image"
                  return 1
            fi
      done

      if ! docker save "${images_array[@]}" -o "$images_tarfile"; then
            echo "Error: Failed to save Docker images to $images_tarfile"
            return 1
      fi

      if ! docker rmi --force "${images_array[@]}"; then
            echo "Error: Failed to remove Docker images"
            return 1
      fi
}

create_directory "$RESOURCE"
download_containerd
download_kubeadm_kubelet
pull_images

log "Download completed successfully!"
