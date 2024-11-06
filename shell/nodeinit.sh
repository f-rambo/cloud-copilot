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

log "Setup completed successfully"
