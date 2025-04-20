#!/bin/bash

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

if [ $(df -k / | awk 'NR==2 {print $4}') -lt 1048576 ]; then
      log "Error: Not enough disk space. At least 1GB free space required."
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

clusterJsonData="{{clusterJsonData}}" # Here is the clusterJsonData from the template variable example: {"name": "test", "region": "ap-southeast-1"}
if [ -z "$clusterJsonData" ]; then
      log "Error: clusterJsonData is empty."
      exit 1
fi

cloudCopilotVersion="v0.0.1"

packageName="cloud-copilot-"$ARCH"-"$cloudCopilotVersion
cloudCopilotPackage=$packageName".tar.gz"

resourceName="resource"
resourceNamePackage=$resourceName"-"$cloudCopilotVersion".tar.gz"

cloudCopilotPackageUrl="https://github.com/f-rambo/cloud-copilot/releases/download/"$cloudCopilotVersion"/"$cloudCopilotPackage
resroucePackageUrl="https://github.com/f-rambo/cloud-copilot/releases/download/v0.0.1/"$resourceNamePackage

function install_cloud_copilot() {
      log "Install cloud copilot..."
      if ! curl -L $cloudCopilotPackageUrl -o $cloudCopilotPackage; then
            log "Error: Failed to download cloud copilot."
            exit 1
      fi
      if ! tar -zxvf $cloudCopilotPackage; then
            log "Error: Failed to extract cloud copilot."
            exit 1
      fi
      if ! sudo cp -r $packageName"/configs" $HOME/; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi

      echo "$clusterJsonData" >$HOME"/cluster.json"

      sed -i "s#cluster_path: \"\"#cluster_path: \"$HOME/cluster.json\"#" $HOME"/configs/config.yaml"

      if ! sudo cp -r $packageName"/shell" $HOME/; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi
      if ! sudo cp -r $packageName"/component" $HOME/; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi
      if ! sudo cp $packageName"/cloud-copilot" /usr/local/bin/cloud-copilot; then
            log "Error: Failed to install cloud copilot."
            exit 1
      fi
      log "Install cloud copilot success."
}

function start_cloud_copilot() {
      log "Starting cloud copilot service..."
      cat <<EOF | sudo tee /etc/systemd/system/cloud-copilot.service
[Unit]
Description=Cloud Copilot Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cloud-copilot
Restart=always
User=$USER

[Install]
WantedBy=multi-user.target
EOF

      if ! sudo systemctl daemon-reload; then
            log "Error: Failed to reload systemd configuration."
            exit 1
      fi

      if ! sudo systemctl start cloud-copilot; then
            log "Error: Failed to start cloud-copilot service."
            exit 1
      fi

      log "Cloud copilot service started successfully."
}

function install_resource() {
      log "Install resource..."
      if [ -d "$HOME/resource" ]; then
            log "Warning: Resource directory already exists. Removing..."
            rm -rf "$HOME/resource"
      fi

      if ! curl -L $resroucePackageUrl -o $resourceNamePackage; then
            log "Error: Failed to download resource."
            exit 1
      fi
      if ! tar -zxvf $resourceNamePackage; then
            log "Error: Failed to extract resource."
            exit 1
      fi
      if ! sudo mv $resourceName "$HOME/"; then
            log "Error: Failed to move resource."
            exit 1
      fi
      log "Install resource success."
}

install_cloud_copilot
start_cloud_copilot
install_resource

log "Install success."
exit 0
