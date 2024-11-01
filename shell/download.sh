#!/bin/bash
set -e

KUBERNETES_VERSION=${1:-"v1.31.2"}
CONTAINERD_VERSION=${2:-"v1.7.23"}
RUNC_VERSION=${3:-"v1.2.0"}
CNIPLUGINS_VERSION=${4:-"v1.6.0"}
OCEAN_VERSION=${5:-"v0.0.1"}
SHIP_VERSION=${6:-"v0.0.1"}
RESOURCE=${7:-"$HOME/resource"}

PLATFORMS=("amd64" "arm64")

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
    echo "Error: Unsupported OS $OS"
    exit 1
fi

create_directory() {
    local dir=$1
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir" || {
            echo "Failed to create directory: $dir"
            exit 1
        }
    fi
}

download_file() {
    local url=$1
    local file=$2
    local checksum_file=$3

    echo "Downloading $url"

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
    sha256sum -c --status <(echo "$(cat $checksum_file)  $file") || {
        echo "SHA256 checksum verification failed for $file"
        exit 1
    }
}

extract_tar() {
    local tarfile=$1
    local dest_dir=$2
    tar -xzf "$tarfile" -C "$dest_dir" || {
        echo "Failed to extract $tarfile"
        exit 1
    }
}

function download_ocean() {
    for platform in "${PLATFORMS[@]}"; do
        echo "download ocean ${OCEAN_VERSION} ${platform}"
        ocean_path="${RESOURCE}/${platform}/ocean/${OCEAN_VERSION}"
        create_directory "$ocean_path"
        ocean_tarfile="linux-${platform}-ocean-${OCEAN_VERSION}.tar.gz"
        if ! download_file "https://github.com/f-rambo/ocean/releases/download/${OCEAN_VERSION}/${ocean_tarfile}" "$ocean_tarfile" "${ocean_tarfile}.sha256sum"; then
            echo "Failed to download file"
            return 1
        fi
        if ! verify_checksum "$ocean_tarfile" "${ocean_tarfile}.sha256sum"; then
            echo "Checksum verification failed"
            rm -f "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
            return 1
        fi
        extract_tar "$ocean_tarfile" "$ocean_path"
        rm -f "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
    done
}

function download_ship() {
    for platform in "${PLATFORMS[@]}"; do
        echo "download ship ${SHIP_VERSION} ${platform}"
        ship_path="${RESOURCE}/${platform}/ship/${SHIP_VERSION}"
        create_directory "$ship_path"
        ship_tarfile="linux-${platform}-ship-${SHIP_VERSION}.tar.gz"
        if ! download_file "https://github.com/f-rambo/ship/releases/download/${SHIP_VERSION}/${ship_tarfile}" "$ship_tarfile" "${ship_tarfile}.sha256sum"; then
            echo "Failed to download file"
            return 1
        fi
        if ! verify_checksum "$ship_tarfile" "${ship_tarfile}.sha256sum"; then
            echo "Checksum verification failed"
            rm -f "$ship_tarfile" "${ship_tarfile}.sha256sum"
            return 1
        fi
        extract_tar "$ship_tarfile" "$ship_path"
        rm -f "$ship_tarfile" "${ship_tarfile}.sha256sum"
    done
}

function download_containerd() {
    for platform in "${PLATFORMS[@]}"; do
        echo "download containerd ${CONTAINERD_VERSION} ${platform}"
        containerd_path="${RESOURCE}/${platform}/containerd/${CONTAINERD_VERSION}"
        create_directory "$containerd_path"
        containerd_tarfile="containerd-${CONTAINERD_VERSION}-linux-${platform}.tar.gz"
        if ! download_file "https://github.com/containerd/containerd/releases/download/${CONTAINERD_VERSION}/${containerd_tarfile}" "${containerd_tarfile}" "${containerd_tarfile}.sha256sum"; then
            echo "Failed to download containerd"
            return 1
        fi
        if ! verify_checksum "$containerd_tarfile" "${containerd_tarfile}.sha256sum"; then
            echo "Checksum verification failed"
            rm -f "$containerd_tarfile" "${containerd_tarfile}.sha256sum"
            return 1
        fi
        extract_tar "$containerd_tarfile" "$containerd_path"
        rm -f "$containerd_tarfile" "${containerd_tarfile}.sha256sum"

        echo "download runc ${RUNC_VERSION} ${platform}"
        runc_path="${RESOURCE}/${platform}/runc/${RUNC_VERSION}"
        create_directory "$runc_path"
        if ! download_file "https://github.com/opencontainers/runc/releases/download/${RUNC_VERSION}/runc.${platform}" "runc.${platform}"; then
            echo "Failed to download runc"
            return 1
        fi
        mv runc.${platform} "$runc_path/runc"

        echo "download cni ${CNI_VERSION} ${platform}"
        cni_path="${RESOURCE}/${platform}/cni-plugins/${CNI_VERSION}"
        create_directory "$cni_path"
        if ! download_file "https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-${platform}-${CNI_VERSION}.tgz" "cni-plugins-linux-${platform}-${CNI_VERSION}.tgz"; then
            echo "Failed to download cni"
            return 1
        fi
        extract_tar "cni-plugins-linux-${platform}-${CNI_VERSION}.tgz" "$cni_path"
        rm -f "cni-plugins-linux-${platform}-${CNI_VERSION}.tgz"
    done
}

function download_kubeadm_kubelet() {
    for platform in "${PLATFORMS[@]}"; do
        echo "download kubeadm kubelet ${KUBERNETES_VERSION} ${platform}"
        kubernetes_path="${RESOURCE}/${platform}/kubernetes/${KUBERNETES_VERSION}"
        create_directory "$kubernetes_path"
        if ! download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${platform}/kubeadm" "kubeadm"; then
            echo "Failed to download kubeadm"
            return 1
        fi
        if ! download_file "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/${platform}/kubelet" "kubelet"; then
            echo "Failed to download kubelet"
            return 1
        fi
        mv kubeadm "$kubernetes_path/kubeadm"
        mv kubelet "$kubernetes_path/kubelet"
    done
}

function download_kubernete_images() {
    local kubeadm_path="${RESOURCE}/${ARCH}/kubernetes/${KUBERNETES_VERSION}/kubeadm"
    if [ ! -f "$kubeadm_path" ]; then
        echo "Error: kubeadm not found"
        return 1
    fi

    if ! chmod 750 "$kubeadm_path"; then
        echo "Error: Failed to change permissions of $kubeadm_path"
        return 1
    fi

    local kube_images=$("$kubeadm_path" config images list --kubernetes-version "$KUBERNETES_VERSION")
    if [ $? -ne 0 ]; then
        echo "Error: Failed to get Kubernetes images list"
        return 1
    fi
    mapfile -t images_array <<<"$kube_images"

    for platform in "${PLATFORMS[@]}"; do
        local images_dir="${RESOURCE}/${platform}/kubernetes/${KUBERNETES_VERSION}/"
        if ! create_directory "$images_dir"; then
            echo "Error: Failed to create directory $images_dir"
            return 1
        fi
        local images_tarfile="${images_dir}/kubernetes-images.tar"

        for image in "${images_array[@]}"; do
            if ! docker pull --platform=linux/$platform "$image"; then
                echo "Error: Failed to pull image $image"
                return 1
            fi
        done

        if ! docker save $(printf "%s" "$kube_images") -o "$images_tarfile"; then
            echo "Error: Failed to save Docker images to $images_tarfile"
            return 1
        fi

        if ! docker rmi -a --force $(printf "%s" "$kube_images"); then
            echo "Error: Failed to remove Docker images"
            return 1
        fi
    done
}

create_directory "$RESOURCE"
download_ocean
download_ship
download_containerd
download_kubeadm_kubelet
download_kubernete_images
