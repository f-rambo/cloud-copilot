#!/bin/bash
set -e

log_file="/var/log/ocean_ship_start.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a $log_file
}

ENV=${1:-"bostionhost"}
OCEAN_VERSION=${2:-"0.0.1"}
SHIP_VERSION=${3:-"0.0.1"}
RESOURCE=${4:-"$HOME/resource"}
SHELL_PATH=${5:-"$HOME/shell"}

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
    mv $RESOURCE/ocean/${OCEAN_VERSION}/${ARCH}/* $OCEAN_PATH/
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
    cat <<EOF > $OCEAN_SYSTEMED_CONF
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
    mv $RESOURCE/ship/${SHIP_VERSION}/${ARCH}/* $SHIP_PATH/
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
    cat <<EOF > $SHIP_SYSTEMED_CONF
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

case $ENV in
  "bostionhost")
    start_ocean
    ;;
  "cluster")
    start_ship
    ;;
esac