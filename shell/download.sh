#!/bin/bash
set -e

KUBERNETES_VERSION=${1:-"v1.31.2"}
CRIO_VERSION=${2:-"v1.31.1"}
OCEAN_VERSION=${3:-"v0.0.1"}
SHIP_VERSION=${4:-"v0.0.1"}
SOFTWARE_RESOURCE_PATH="./resource/kubernetes-software/"
OCEAN_RESOURCE_PATH="./resource/ocean/"
SHIP_RESOURCE_PATH="./resource/ship/"

PLATFORMS=("amd64" "arm64")

create_directory() {
    local dir=$1
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir" || { echo "Failed to create directory: $dir"; exit 1; }
    fi
}

download_file() {
    local url=$1
    local file=$2
    local checksum_file=$3

    if [[ -z "$url" || -z "$file" ]]; then
        echo "Error: URL and file parameters are required."
        exit 1
    fi

    if [ ! -f "$file" ]; then
        if ! curl -L -C - --fail -O "$url"; then
            echo "Failed to download $url"
            rm -f "$file"
            exit 1
        fi

        if [ -n "$checksum_file" ]; then
            if ! curl -L -C - --fail -O "$url.sha256sum"; then
                echo "Failed to download $url.sha256sum"
                rm -f "$file"
                exit 1
            fi

        fi
    fi
}

verify_checksum() {
    local file=$1
    local checksum_file=$2
    sha256sum -c --status <(echo "$(cat $checksum_file)  $file") || { echo "SHA256 checksum verification failed for $file"; exit 1; }
}

extract_tar() {
    local tarfile=$1
    local dest_dir=$2
    tar -xzf "$tarfile" -C "$dest_dir" || { echo "Failed to extract $tarfile"; exit 1; }
}

function install_ocean() {
    for platform in "${PLATFORMS[@]}"; do
        echo "install ocean ${OCEAN_VERSION} ${platform}"
        ocean_path="${OCEAN_RESOURCE_PATH}${OCEAN_VERSION}/${platform}"
        create_directory "$ocean_path"
        ocean_tarfile="linux-${platform}-ocean-${OCEAN_VERSION}.tar.gz"
        download_file "https://github.com/f-rambo/ocean/releases/download/${OCEAN_VERSION}/${ocean_tarfile}" "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
        verify_checksum "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
        extract_tar "$ocean_tarfile" "$ocean_path"
        rm -f "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
    done
}

function install_ship() {
    for platform in "${PLATFORMS[@]}"; do
        echo "install ship ${SHIP_VERSION} ${platform}"
        ship_path="${SHIP_RESOURCE_PATH}${SHIP_VERSION}/${platform}"
        create_directory "$ship_path"
        ship_tarfile="linux-${platform}-ship-${SHIP_VERSION}.tar.gz"
        download_file "https://github.com/f-rambo/ship/releases/download/${SHIP_VERSION}/${ship_tarfile}" "$ship_tarfile" "${ship_tarfile}.sha256sum"
        verify_checksum "$ship_tarfile" "${ship_tarfile}.sha256sum"
        extract_tar "$ship_tarfile" "$ship_path"
        rm -f "$ship_tarfile" "${ship_tarfile}.sha256sum"
    done
}

function install_crio() {
    for platform in "${PLATFORMS[@]}"; do
        echo "install crio ${CRIO_VERSION} ${platform}"
        crio_path="${SOFTWARE_RESOURCE_PATH}${KUBERNETES_VERSION}/crio/${platform}"
        create_directory "$crio_path"
        criotarfile="cri-o.${platform}.${CRIO_VERSION}.tar.gz"
        download_file "https://storage.googleapis.com/cri-o/artifacts/${criotarfile}" "$criotarfile" "${criotarfile}.sha256sum"
        verify_checksum "$criotarfile" "${criotarfile}.sha256sum"
        extract_tar "$criotarfile" "$crio_path"
        rm -f "$criotarfile" "${criotarfile}.sha256sum"
    done
}

function install_kubeadm_kubelet() {
    for platform in "${PLATFORMS[@]}"; do
        echo "install kubeadm kubelet ${KUBERNETES_VERSION} ${platform}"
        kubernetes_path="${SOFTWARE_RESOURCE_PATH}${KUBERNETES_VERSION}/kubernetes/${platform}"
        create_directory "$kubernetes_path"
        download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${platform}/kubeadm" "kubeadm"
        download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${platform}/kubelet" "kubelet"
        mv kubeadm "$kubernetes_path/kubeadm" || { echo "Failed to move kubeadm"; exit 1; }
        mv kubelet "$kubernetes_path/kubelet" || { echo "Failed to move kubelet"; exit 1; }
    done
}

create_directory "$OCEAN_RESOURCE_PATH"
install_ocean
create_directory "$SHIP_RESOURCE_PATH"
install_ship
create_directory "$SOFTWARE_RESOURCE_PATH"
install_crio
install_kubeadm_kubelet