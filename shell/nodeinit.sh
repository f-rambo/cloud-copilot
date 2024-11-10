#!/bin/bash
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
