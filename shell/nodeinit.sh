#!/bin/bash
set -e

LOG_FILE="/var/log/hostname_and_iptables_setup.log"

log() {
    local message="$1"
    echo "$(date +'%Y-%m-%d %H:%M:%S') - $message" | tee -a $LOG_FILE
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

log "Enabling bridge-nf-call-iptables"
if ! sysctl -w net.bridge.bridge-nf-call-iptables=1; then
    log "Error: Failed to enable bridge-nf-call-iptables."
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

log "Flushing iptables rules"
if ! iptables -F && ! iptables -t nat -F; then
    log "Error: Failed to flush iptables rules."
    exit 1
fi

log "Adding iptables rules"
if ! iptables -A INPUT -i lo -j ACCEPT ||
    ! iptables -A INPUT -p tcp --dport 22 -j ACCEPT ||
    ! iptables -A INPUT -p tcp --dport 6443 -j ACCEPT ||
    ! iptables -A INPUT -p tcp --dport 10250 -j ACCEPT ||
    ! iptables -A INPUT -p tcp --dport 10256 -j ACCEPT ||
    ! iptables -A INPUT -p tcp --match multiport --dports 30000:32767 -j ACCEPT ||
    ! iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT ||
    ! iptables -A INPUT -j DROP; then
    log "Error: Failed to add iptables rules."
    exit 1
fi

log "Saving iptables rules"
if ! iptables-save | tee /etc/iptables/rules.v4; then
    log "Error: Failed to save iptables rules."
    exit 1
fi

log "Setup completed successfully"
