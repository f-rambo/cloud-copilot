#!/bin/bash
set -e

if [ -n "$SUDO_USER" ]; then
      ORIGINAL_USER=$SUDO_USER
      ORIGINAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
      ORIGINAL_USER=$USER
      ORIGINAL_HOME=$HOME
fi

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
resourcePackageUrl="https://github.com/f-rambo/cloud-copilot/releases/download/"$cloudCopilotVersion"/"$resourceNamePackage

function install_cloud_copilot() {
      log "Install cloud copilot..."
      if [ -d "$packageName" ]; then
            log "Package directory $packageName already exists, skipping download and extraction."
      else
            if ! curl -L $cloudCopilotPackageUrl -o $cloudCopilotPackage; then
                  log "Error: Failed to download cloud copilot."
                  exit 1
            fi
            if ! tar -zxvf $cloudCopilotPackage; then
                  log "Error: Failed to extract cloud copilot."
                  exit 1
            fi
      fi
      if ! cp -r $packageName"/configs" "$ORIGINAL_HOME/"; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi

      echo "$clusterJsonData" >"$ORIGINAL_HOME/cluster.json"

      sed -i "s#cluster: \"\"#cluster: \"$ORIGINAL_HOME/cluster.json\"#" "$ORIGINAL_HOME/configs/config.yaml"

      if ! cp -r $packageName"/shell" "$ORIGINAL_HOME/"; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi
      if ! cp -r $packageName"/component" "$ORIGINAL_HOME/"; then
            log "Error: Failed to move cloud copilot."
            exit 1
      fi
      if ! cp $packageName"/cloud-copilot" /usr/local/bin/cloud-copilot; then
            log "Error: Failed to install cloud copilot."
            exit 1
      fi
      log "Install cloud copilot success."
}

function start_cloud_copilot() {
      log "Starting cloud copilot service..."
      cat <<EOF >/etc/systemd/system/cloud-copilot.service
[Unit]
Description=Cloud Copilot Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cloud-copilot -conf $ORIGINAL_HOME/configs
WorkingDirectory=$ORIGINAL_HOME
Restart=always
User=$ORIGINAL_USER

[Install]
WantedBy=multi-user.target
EOF

      if ! systemctl daemon-reload; then
            log "Error: Failed to reload systemd configuration."
            exit 1
      fi

      if ! systemctl start cloud-copilot; then
            log "Error: Failed to start cloud-copilot service."
            exit 1
      fi

      log "Cloud copilot service started successfully."
}

function install_resource() {
      log "Install resource..."
      if [ -d "$ORIGINAL_HOME/$resourceName" ]; then
            log "Resource directory already exists."
            return 0
      fi

      if ! curl -L "$resourcePackageUrl" -o "$resourceNamePackage"; then
            log "Error: Failed to download resource."
            exit 1
      fi
      if ! tar -zxvf "$resourceNamePackage"; then
            log "Error: Failed to extract resource."
            exit 1
      fi
      if ! mv "$resourceName" "$ORIGINAL_HOME/"; then
            log "Error: Failed to move resource."
            exit 1
      fi
      log "Install resource success."
}

install_cloud_copilot
start_cloud_copilot
install_resource

exit 0
