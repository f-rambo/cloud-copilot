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

log "Checking and enabling IP forwarding if needed"
current_value=$(cat /proc/sys/net/ipv4/ip_forward)
if [ "$current_value" != "1" ]; then
    if ! sysctl -w net.ipv4.ip_forward=1; then
        log "Error: Failed to enable IP forwarding."
        exit 1
    fi
    log "IP forwarding has been enabled"
else
    log "IP forwarding is already enabled"
fi

log "Checking swap status"
if [ "$(swapon --show)" ]; then
    log "Disabling swap"
    if ! swapoff -a; then
        log "Error: Failed to disable swap."
        exit 1
    fi
    log "Swap has been disabled"
else
    log "Swap is already disabled"
fi

log "Commenting out swap in /etc/fstab"
log "Checking and updating swap entries in /etc/fstab"
if grep -q "^[^#].*[ ]swap[ ]" /etc/fstab; then
    log "Found uncommented swap entry, commenting it out"
    if ! sed -i '/ swap / s/^\([^#]\)/#\1/' /etc/fstab; then
        log "Error: Failed to comment out swap in /etc/fstab."
        exit 1
    fi
    log "Successfully commented out swap entry in /etc/fstab"
else
    log "No uncommented swap entries found in /etc/fstab"
fi

log "Checking and installing required packages"
if command -v apt-get &>/dev/null; then
    # 更新包列表
    if ! apt-get update; then
        log "Error: Failed to update package list."
        exit 1
    fi

    # 检查并安装 conntrack
    if ! command -v conntrack &>/dev/null; then
        log "Installing conntrack"
        if ! apt-get install -y conntrack; then
            log "Error: Failed to install conntrack."
            exit 1
        fi
    else
        log "conntrack is already installed"
    fi

    # 检查并安装 yq
    if ! command -v yq &>/dev/null; then
        log "Installing yq"
        if ! apt-get install -y yq; then
            log "Error: Failed to install yq."
            exit 1
        fi
    else
        log "yq is already installed"
    fi

    # 检查并安装 jq
    if ! command -v jq &>/dev/null; then
        log "Installing jq"
        if ! apt-get install -y jq; then
            log "Error: Failed to install jq."
            exit 1
        fi
    else
        log "jq is already installed"
    fi

    # 检查并安装 lvm2
    if ! command -v lvm2 &>/dev/null; then
        log "Installing lvm2"
        if ! apt-get install -y lvm2; then
            log "Error: Failed to install lvm2."
            exit 1
        fi
    else
        log "lvm2 is already installed"
    fi

    # 检查并安装 util-linux (包含 lsblk)
    if ! command -v lsblk &>/dev/null; then
        log "Installing util-linux"
        if ! apt-get install -y util-linux; then
            log "Error: Failed to install util-linux."
            exit 1
        fi
    else
        log "util-linux (lsblk) is already installed"
    fi
elif command -v yum &>/dev/null; then
    # 更新包列表
    if ! yum update; then
        log "Error: Failed to update package list."
        exit 1
    fi

    # 检查并安装 conntrack
    if ! command -v conntrack &>/dev/null; then
        log "Installing conntrack"
        if ! yum install -y conntrack; then
            log "Error: Failed to install conntrack."
            exit 1
        fi
    else
        log "conntrack is already installed"
    fi

    # 检查并安装 yq
    if ! command -v yq &>/dev/null; then
        log "Installing yq"
        if ! yum install -y yq; then
            log "Error: Failed to install yq."
            exit 1
        fi
    else
        log "yq is already installed"
    fi

    # 检查并安装 jq
    if ! command -v jq &>/dev/null; then
        log "Installing jq"
        if ! yum install -y jq; then
            log "Error: Failed to install jq."
            exit 1
        fi
    else
        log "jq is already installed"
    fi

    # 检查并安装 lvm2
    if ! command -v lvm2 &>/dev/null; then
        log "Installing lvm2"
        if ! yum install -y lvm2; then
            log "Error: Failed to install lvm2."
            exit 1
        fi
    else
        log "lvm2 is already installed"
    fi

    # 检查并安装 util-linux (包含 lsblk)
    if ! command -v lsblk &>/dev/null; then
        log "Installing util-linux"
        if ! yum install -y util-linux; then
            log "Error: Failed to install util-linux."
            exit 1
        fi
    else
        log "util-linux (lsblk) is already installed"
    fi
elif command -v dnf &>/dev/null; then
    # 更新包列表
    if ! dnf update; then
        log "Error: Failed to update package list."
        exit 1
    fi

    # 检查并安装 conntrack
    if ! command -v conntrack &>/dev/null; then
        log "Installing conntrack"
        if ! dnf install -y conntrack; then
            log "Error: Failed to install conntrack."
            exit 1
        fi
    else
        log "conntrack is already installed"
    fi

    # 检查并安装 yq
    if ! command -v yq &>/dev/null; then
        log "Installing yq"
        if ! dnf install -y yq; then
            log "Error: Failed to install yq."
            exit 1
        fi
    else
        log "yq is already installed"
    fi

    # 检查并安装 jq
    if ! command -v jq &>/dev/null; then
        log "Installing jq"
        if ! dnf install -y jq; then
            log "Error: Failed to install jq."
            exit 1
        fi
    else
        log "jq is already installed"
    fi

    # 检查并安装 lvm2
    if ! command -v lvm2 &>/dev/null; then
        log "Installing lvm2"
        if ! dnf install -y lvm2; then
            log "Error: Failed to install lvm2."
            exit 1
        fi
    else
        log "lvm2 is already installed"
    fi

    # 检查并安装 util-linux (包含 lsblk)
    if ! command -v lsblk &>/dev/null; then
        log "Installing util-linux"
        if ! dnf install -y util-linux; then
            log "Error: Failed to install util-linux."
            exit 1
        fi
    else
        log "util-linux (lsblk) is already installed"
    fi
else
    log "Error: Unsupported package manager."
    exit 1
fi

log "Setup completed successfully"
