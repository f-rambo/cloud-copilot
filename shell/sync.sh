#!/bin/bash
set -e


SERVER_IP=$1
SERVER_PORT=$2
SERVER_USER=$3
PRIVATE_KEY=$4
OCEAN_DATA=${5:-"$HOME/.ocean"}
SHIP_DATA=${6:-"$HOME/.ship"}
RESOURCE=${7:-"$HOME/resource"}
SHELL_PATH=${8:-"$HOME/shell"}
PRIVATE_KEY_PATH="/tmp/private_key"

echo "$PRIVATE_KEY" > $PRIVATE_KEY_PATH && chmod 600 $PRIVATE_KEY_PATH

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

    if [ ! -d "$SHIP_DATA" ]; then
        log "Ship Data directory does not exist"
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
    if [ -d "$SHIP_DATA" ]; then
        cp -r $SHIP_DATA/* /tmp/data_resource/
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
    ssh -i $PRIVATE_KEY_PATH -p $SERVER_PORT $SERVER_USER@$SERVER_IP "tar -xzf /tmp/data_resource.tar.gz -C /tmp"
    log "Data resource extracted successfully."
    log "Data resource extract path: /tmp/data_resource"
}

function move_files() {
    local source_dir=$1
    local target_dir=$2
    local target_user=$3
    
    if [ -d "$source_dir" ]; then
        ssh -i $PRIVATE_KEY_PATH -p $SERVER_PORT $target_user@$SERVER_IP "mkdir -p $target_dir && mv /tmp/data_resource/$(basename $source_dir) $target_dir"
        log "$(basename $source_dir) moved successfully."
        log "Move path: $target_dir"
    fi
}

function mvfile() {
    log "Moving files..."
    move_files $RESOURCE /home/$SERVER_USER/resource $SERVER_USER
    move_files $OCEAN_DATA /home/$SERVER_USER/.ocean $SERVER_USER
    move_files $SHIP_DATA /home/$SERVER_USER/.ship $SERVER_USER
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