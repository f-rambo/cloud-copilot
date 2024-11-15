package utils

const (
	Download   = "download.sh"
	Kubernetes = "kubernetes.sh"
	NodeInit   = "nodeinit.sh"
	Service    = "service.sh"
	Sync       = "sync.sh"
	SystemInfo = "systeminfo.sh"
)

const (
	DownloadShell = `#!/bin/bash
set -e

RESOURCE=${1:-"$HOME/resource"}
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

    if [[ ! -f "$checksum_file" ]]; then
        log "Checksum file not found: $checksum_file" >&2
        return 1
    fi

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

function download_ocean() {
    log "download ocean ${SERVICE_VERSION} ${ARCH}"
    ocean_path="${RESOURCE}/ocean/${SERVICE_VERSION}"
    create_directory "$ocean_path"
    ocean_tarfile="linux-${ARCH}-ocean-${SERVICE_VERSION}.tar.gz"
    if ! download_file "https://github.com/f-rambo/ocean/releases/download/${SERVICE_VERSION}/${ocean_tarfile}" "$ocean_tarfile" "${ocean_tarfile}.sha256sum"; then
        log "Failed to download file"
        return 1
    fi
    if ! verify_checksum "$ocean_tarfile" "${ocean_tarfile}.sha256sum"; then
        log "Checksum verification failed"
        rm -f "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
        return 1
    fi
    extract_tar "$ocean_tarfile" "$ocean_path"
    rm -f "$ocean_tarfile" "${ocean_tarfile}.sha256sum"
}

function download_ship() {
    log "download ship ${SERVICE_VERSION} ${ARCH}"
    ship_path="${RESOURCE}/ship/${SERVICE_VERSION}"
    create_directory "$ship_path"
    ship_tarfile="linux-${ARCH}-ship-${SERVICE_VERSION}.tar.gz"
    if ! download_file "https://github.com/f-rambo/ship/releases/download/${SERVICE_VERSION}/${ship_tarfile}" "$ship_tarfile" "${ship_tarfile}.sha256sum"; then
        log "Failed to download file"
        return 1
    fi
    if ! verify_checksum "$ship_tarfile" "${ship_tarfile}.sha256sum"; then
        log "Checksum verification failed"
        rm -f "$ship_tarfile" "${ship_tarfile}.sha256sum"
        return 1
    fi
    extract_tar "$ship_tarfile" "$ship_path"
    rm -f "$ship_tarfile" "${ship_tarfile}.sha256sum"
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
download_ocean
download_ship
download_containerd
download_kubeadm_kubelet
pull_images

log "Download completed successfully!"

`
	KubernetesShell = `#!/bin/bash
set -e

log() {
  local message="$1"
  echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

RESOURCE=${1:-"$HOME/resource"}
KUBERNETES_VERSION=${2:-"v1.31.2"}
CONTAINERD_VERSION=${3:-"v2.0.0"}
RUNC_VERSION=${4:-"v1.2.1"}

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

if ss -tulpn | grep -q ":6443"; then
  log "Error: Port 6443 is already in use. Please free up the port and try again."
  exit 1
fi

kubeletService=$(
  cat <<EOF
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs/
Wants=network-online.target
After=network-online.target

[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
EnvironmentFile=-/etc/sysconfig/kubelet
ExecStart=/usr/local/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
)

function install_kubernetes_software() {
  kubernetesPath="$RESOURCE/$ARCH/kubernetes/$KUBERNETES_VERSION"

  if [ ! -d "$kubernetesPath" ] || [ ! -r "$kubernetesPath" ]; then
    log "Error: Directory $kubernetesPath does not exist or is not readable"
    exit 1
  fi

  if [ ! -f "$kubernetesPath/kubeadm" ]; then
    log "Error: File $kubernetesPath/kubeadm does not exist"
    exit 1
  fi

  if ! install -m 755 "$kubernetesPath/kubeadm" /usr/local/bin/kubeadm; then
    log "Error: Failed to install kubeadm"
    exit 1
  fi

  if [ ! -f "$kubernetesPath/kubelet" ]; then
    log "Error: File $kubernetesPath/kubelet does not exist"
    exit 1
  fi

  if ! install -m 755 "$kubernetesPath/kubelet" /usr/local/bin/kubelet; then
    log "Error: Failed to install kubelet"
    exit 1
  fi

  if ! echo "$kubeletService" | tee /usr/lib/systemd/system/kubelet.service >/dev/null; then
    log "Error: Failed to write to /usr/lib/systemd/system/kubelet.service"
    exit 1
  fi

}

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
  log "install containerd..."

  containerdPath="$RESOURCE/$ARCH/containerd/$CONTAINERD_VERSION"
  if [ ! -d "$containerdPath" ] || [ ! -r "$containerdPath" ]; then
    log "Error: Directory $containerdPath does not exist or is not readable"
    exit 1
  fi

  chmod -R 755 "${containerdPath}/bin/"

  cp -r $containerdPath/bin/* /usr/local/bin/

  if ! ctr --version; then
    log "Error: Failed to start containerd service"
    exit 1
  fi

  mkdir -p /etc/containerd
  touch /etc/containerd/config.toml
  containerd config default | sed -e "s/SystemdCgroup = false/SystemdCgroup = true/g" | tee /etc/containerd/config.toml

  if ! echo "$containerdService" | tee /usr/lib/systemd/system/containerd.service >/dev/null; then
    log "Error: Failed to write to /usr/lib/systemd/system/containerd.service"
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

  log "install runc..."
  runcPath="$RESOURCE/$ARCH/runc/$RUNC_VERSION"
  if [ ! -d "$runcPath" ] || [ ! -r "$runcPath" ]; then
    log "Error: Directory $runcPath does not exist or is not readable"
    exit 1
  fi

  install -m 755 "$runcPath/runc" /usr/local/bin/runc

  # create namespace for containerd
  if ! ctr namespace list | grep -q "k8s.io"; then
    ctr namespace create k8s.io
  fi

  # import pause image
  kubernetes_image_path="$RESOURCE/$ARCH/kubernetes/$KUBERNETES_VERSION/kubernetes-images.tar"
  if [ ! -f "$kubernetes_image_path" ] || [ ! -r "$kubernetes_image_path" ]; then
    log "Error: File $kubernetes_image_path does not exist or is not readable"
    exit 1
  fi

  ctr -n k8s.io images import "$kubernetes_image_path"
}

install_kubernetes_software

if systemctl is-active --quiet containerd; then
  log "containerd is already running, skipping installation."
else
  log "containerd is not running, proceeding with installation."
  install_containerd
fi

log "kubernetes software installation completed successfully."

exit 0

`
	NodeInitShell = `#!/bin/bash
set -e

log() {
    local message="$1"
    echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

if [ -z "$1" ]; then
    log "Error: Hostname is required."
    exit 1
fi

HOMSNAME=$1

log "Setting hostname to $HOMSNAME"
if ! hostnamectl set-hostname $HOMSNAME; then
    log "Error: Failed to set hostname."
    exit 1
fi

log "Checking if $HOMSNAME already exists in /etc/hosts"
if grep -q " $HOMSNAME$" /etc/hosts; then
    log "$HOMSNAME already exists in /etc/hosts."
else
    log "Adding $HOMSNAME to /etc/hosts"
    if ! echo "127.0.0.1 $HOMSNAME" >>/etc/hosts; then
        log "Error: Failed to add $HOMSNAME to /etc/hosts."
        exit 1
    fi
fi

log "Enabling IP forwarding"
if ! sysctl -w net.ipv4.ip_forward=1; then
    log "Error: Failed to enable IP forwarding."
    exit 1
fi

log "Disabling swap"
if ! swapoff -a; then
    log "Error: Failed to disable swap."
    exit 1
fi

log "Commenting out swap in /etc/fstab"
if ! sed -i '/ swap / s/^/#/' /etc/fstab; then
    log "Error: Failed to comment out swap in /etc/fstab."
    exit 1
fi

log "Installing conntrack"
if command -v apt-get &>/dev/null; then
    if ! apt-get update && apt-get install -y conntrack; then
        log "Error: Failed to install conntrack."
        exit 1
    fi
elif command -v yum &>/dev/null; then
    if ! yum install -y conntrack; then
        log "Error: Failed to install conntrack."
        exit 1
    fi
elif command -v dnf &>/dev/null; then
    if ! dnf install -y conntrack; then
        log "Error: Failed to install conntrack."
        exit 1
    fi
else
    log "Error: Unsupported package manager."
    exit 1
fi

log "Setup completed successfully"

`
	ServiceShell = `#!/bin/bash
set -e

log_file="/var/log/ocean_ship_start.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a $log_file
}

SERVICE=$1
ENV=$2
VERSION=${3:-"0.0.1"}
RESOURCE=${5:-"$HOME/resource"}
SHELL_PATH=${6:-"$HOME/shell"}

OCEAN_PATH="$HOME/app/ocean"
SHIP_PATH="$HOME/app/ship"

ARCH=$(uname -m)
case $ARCH in
aarch64)
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

function start_ocean() {
    if [ ! -d "$OCEAN_PATH" ]; then
        mkdir -p "$OCEAN_PATH"
    fi
    if [ ! -w "$OCEAN_PATH" ]; then
        echo "Error: No write permission for $OCEAN_PATH"
        exit 1
    fi
    mv $RESOURCE/ocean/${VERSION}/${ARCH}/* $OCEAN_PATH/
    if [ ! -f "$OCEAN_PATH/configs/config.yaml" ]; then
        echo "Error: Config file $OCEAN_PATH/configs/config.yaml not found"
        exit 1
    fi
    sed -i 's/^  env: .*/  env: $ENV/' $OCEAN_PATH/configs/config.yaml
    sed -i 's/^  shell: .*/  shell: $SHELL_PATH/' $OCEAN_PATH/configs/config.yaml
    sed -i 's/^  resource: .*/  resource: $RESOURCE/' $OCEAN_PATH/configs/config.yaml
    OCEAN_SYSTEMED_CONF="/etc/systemd/system/ocean.service"
    if [ ! -w "/etc/systemd/system" ]; then
        echo "Error: No write permission for /etc/systemd/system"
        exit 1
    fi
    cat <<EOF >$OCEAN_SYSTEMED_CONF
[Unit]
Description=Ocean Service
After=network.target

[Service]
User=$USER
ExecStart=$OCEAN_PATH/bin/ocean -conf $OCEAN_PATH/configs/config.yaml
Restart=on-failure
WorkingDirectory=$OCEAN_PATH

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl start ocean
}

function start_ship() {
    if [ ! -d "$SHIP_PATH" ]; then
        mkdir -p "$SHIP_PATH"
    fi
    if [ ! -w "$SHIP_PATH" ]; then
        echo "Error: No write permission for $SHIP_PATH"
        exit 1
    fi
    mv $RESOURCE/ship/${VERSION}/${ARCH}/* $SHIP_PATH/
    if [ ! -f "$SHIP_PATH/configs/config.yaml" ]; then
        echo "Error: Config file $SHIP_PATH/configs/config.yaml not found"
        exit 1
    fi
    sed -i 's/^  env: .*/  env: $ENV/' $SHIP_PATH/configs/config.yaml
    sed -i 's/^  shell: .*/  shell: $SHELL_PATH/' $SHIP_PATH/configs/config.yaml
    sed -i 's/^  resource: .*/  resource: $RESOURCE/' $SHIP_PATH/configs/config.yaml
    SHIP_SYSTEMED_CONF="/etc/systemd/system/ship.service"
    if [ ! -w "/etc/systemd/system" ]; then
        echo "Error: No write permission for /etc/systemd/system"
        exit 1
    fi
    cat <<EOF >$SHIP_SYSTEMED_CONF
[Unit]
Description=Ship Service
After=network.target

[Service]
User=$USER
ExecStart=$SHIP_PATH/bin/ship -conf $SHIP_PATH/configs/config.yaml
Restart=on-failure
WorkingDirectory=$SHIP_PATH

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl start ship
}

case $SERVICE in
"ocean")
    start_ocean
    ;;
"ship")
    start_ship
    ;;
esac

`
	SyncShell = `#!/bin/bash
set -e

SERVER_IP=$1
SERVER_PORT=$2
SERVER_USER=$3
PRIVATE_KEY=$4
OCEAN_DATA=${5:-"$HOME/.ocean"}
RESOURCE=${6:-"$HOME/resource"}
SHELL_PATH=${7:-"$HOME/shell"}
PRIVATE_KEY_PATH="/tmp/private_key"

echo "$PRIVATE_KEY" >$PRIVATE_KEY_PATH && chmod 600 $PRIVATE_KEY_PATH

LOG_FILE="/var/log/data_sync.log"

function log() {
    local message=$1
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $message" | tee -a $LOG_FILE
}

function verify_params() {
    if [ -z "$SERVER_IP" ]; then
        log "Server IP is required"
        exit 1
    fi

    if [ -z "$SERVER_PORT" ]; then
        log "Server Port is required"
        exit 1
    fi

    if [ -z "$SERVER_USER" ]; then
        log "Server User is required"
        exit 1
    fi

    if [ -z "$PRIVATE_KEY" ]; then
        log "Private Key is required"
        exit 1
    fi

    if [ ! -f "$PRIVATE_KEY" ]; then
        log "Private Key file does not exist"
        exit 1
    fi

    if [ ! -d "$OCEAN_DATA" ]; then
        log "Ocean Data directory does not exist"
        exit 1
    fi

    if [ ! -d "$RESOURCE" ]; then
        log "Resource directory does not exist"
        exit 1
    fi
}

function package_data_resource() {
    log "Packaging data resource..."
    mkdir /tmp/data_resource
    if [ -d "$RESOURCE" ]; then
        cp -r $RESOURCE/* /tmp/data_resource/
    fi
    if [ -d "$SHELL_PATH" ]; then
        cp -r $SHELL_PATH/* /tmp/data_resource/
    fi
    if [ -d "$OCEAN_DATA" ]; then
        cp -r $OCEAN_DATA/* /tmp/data_resource/
    fi
    tar -czvf /tmp/data_resource.tar.gz -C /tmp/data_resource .
    rm -rf /tmp/data_resource
    log "Data resource packaged successfully."
    log "Data resource package path: /tmp/data_resource.tar.gz"
}

function sync_data_resource() {
    log "Syncing data resource..."
    rsync -avz -e "ssh -i $PRIVATE_KEY_PATH -p $SERVER_PORT" /tmp/data_resource.tar.gz $SERVER_USER@$SERVER_IP:/tmp/data_resource.tar.gz
    log "Data resource synced successfully."
}

function extract_tar() {
    log "Extracting data resource..."
    ssh -i $PRIVATE_KEY_PATH -p $SERVER_PORT $SERVER_USER@$SERVER_IP "tar -xzf /tmp/data_resource.tar.gz -C $HOME"
    log "Data resource extracted successfully."
    log "Data resource extract path: /tmp/data_resource"
    rm /tmp/data_resource.tar.gz
}

function move_files() {
    local source_dir=$1
    local target_dir=$2
    local target_user=$3

    if [ -d "$source_dir" ]; then
        ssh -i $PRIVATE_KEY_PATH -p $SERVER_PORT $target_user@$SERVER_IP "mkdir -p $target_dir && mv $HOME/data_resource/$(basename $source_dir) $target_dir"
        log "$(basename $source_dir) moved successfully."
        log "Move path: $target_dir"
    fi
}

function mvfile() {
    log "Moving files..."
    move_files $RESOURCE /home/$SERVER_USER/resource $SERVER_USER
    move_files $OCEAN_DATA /home/$SERVER_USER/.ocean $SERVER_USER
    move_files $SHELL_PATH /home/$SERVER_USER/shell $SERVER_USER
}

function handle_error() {
    local error_code=$?
    log "An error occurred with code $error_code. Exiting..."
    exit $error_code
}

trap handle_error ERR

verify_params
package_data_resource
sync_data_resource
mvfile

`
	SystemInfoShell = `#!/bin/bash

uuid=$(sudo dmidecode -s system-uuid)

os=$(uname -s)

arch=$(uname -m)
case $arch in
aarch64)
      arch="arm64"
      ;;
x86_64)
      arch="amd64"
      ;;
esac

memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
memory_gb=$(((memory_kb + 1048575) / 1048576))

cpu_cores=$(nproc)

if command -v nvidia-smi &>/dev/null; then
      gpu_count=$(nvidia-smi --query-gpu=name --format=csv,noheader | wc -l)
      gpu_info=$(nvidia-smi --query-gpu=name --format=csv,noheader)
else
      gpu_info="No NVIDIA GPU found"
      gpu_count=0
fi

total_disk_bytes=0
while IFS= read -r line; do
      # Ignore the header line
      if [[ "$line" == "Size" ]]; then
            continue
      fi

      # Extract size and unit
      size=$(echo $line | awk '{print $1}')
      unit=${size: -1}
      num=${size%?}

      # Remove decimal part if it exists
      num=$(echo "$num" | cut -d'.' -f1)

      # Convert size to bytes
      case $unit in
      K)
            size_bytes=$((num * 1024))
            ;;
      M)
            size_bytes=$((num * 1024 * 1024))
            ;;
      G)
            size_bytes=$((num * 1024 * 1024 * 1024))
            ;;
      T)
            size_bytes=$((num * 1024 * 1024 * 1024 * 1024))
            ;;
      *)
            size_bytes=0
            ;;
      esac

      total_disk_bytes=$((total_disk_bytes + size_bytes))
done < <(df -h --output=size | tail -n +2)

total_disk_gb=$(((total_disk_bytes + 1073741823) / 1073741824))

inner_ip=$(hostname -I | awk '{print $1}')

json_output=$(
      cat <<EOF
{
  "id": "$uuid",
  "os": "$os",
  "arch": "$arch",
  "mem": "${memory_gb}",
  "cpu": "$cpu_cores",
  "gpu": "$gpu_count",
  "gpu_info": "$gpu_info",
  "disk": "${total_disk_gb}",
  "inner_ip": "$inner_ip"
}
EOF
)

echo "$json_output"

`
)

var shellMap = map[string]string{
	Download:   DownloadShell,
	Kubernetes: KubernetesShell,
	NodeInit:   NodeInitShell,
	Service:    ServiceShell,
	Sync:       SyncShell,
	SystemInfo: SystemInfoShell,
}

func ShellToolsInit() error {
	shellPackagePath, err := GetServerStorePathByNames(ShellPackage)
	if err != nil {
		return err
	}
	for fileName, shell := range shellMap {
		err = WriteFile(shellPackagePath, fileName, shell)
		if err != nil {
			return err
		}
	}
	return nil
}
