package infrastructure

import (
	"fmt"
)

func getInstallScriptAndStartOcean(oceanPath, shipPath, scriptEnv string) string {
	return fmt.Sprintf(`#!/bin/bash

SYSTEM=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$1
OCEAN_VERSION=$2
SHIP_VERSION=$3
OCEAN_NAME="ocean"
SHIP_NAME="ship"

echo SYSTEM: $SYSTEM
echo ARCH: $ARCH
echo OCEAN_VERSION: $OCEAN_VERSION
echo SHIP_VERSION: $SHIP_VERSION

# linux/amd64 linux/arm64 linux/riscv64 linux/ppc64le
# Define supported platforms
PLATFORMS=("amd64" "arm64")

# Check if the provided ARCH is supported
is_supported=false
for platform in "${PLATFORMS[@]}"; do
    if [[ "$platform" == "$ARCH" ]]; then
        is_supported=true
        break
    fi
done

if [ "$is_supported" = false ]; then
    echo "Unsupported architecture: $ARCH"
    echo "Supported platforms are: ${PLATFORMS[*]}"
    exit 1
fi

# Check if versions are provided
if [ -z "$OCEAN_VERSION" ] || [ -z "$SHIP_VERSION" ]; then
    echo "Usage: $0 <ARCH> <OCEAN_VERSION> <SHIP_VERSION>"
    exit 1
fi

install_tools() {
    if [ -f /etc/debian_version ]; then
        apt update
        apt install -y curl tar file net-tools || { echo "Failed to install tools"; exit 1; }
    elif [ -f /etc/redhat-release ]; then
        yum update -y
        yum install -y curl tar file net-tools || { echo "Failed to install tools"; exit 1; }
    else
        echo "unknown system type"
        exit 1
    fi
}

install_tools

# Check if URLs are valid
check_url() {
    local url=$1
    if ! curl -f -s -I "$url"; then
        echo "Invalid URL: $url"
        exit 1
    fi
}

# Function to download and extract files
download_and_extract() {
    local platform=$1
    local type=$2
    local url=$3
    local target_dir=$4
    local repo_path=$5

    check_url $url

    if [ "$type" == "ocean" ]; then
        echo "Downloading ocean platform $platform from $url"
    elif [ "$type" == "ship" ]; then
        echo "Downloading ship platform $platform from $url"
    else
        echo "Unknown type: $type"
        exit 1
    fi

    curl -L $url -o $repo_path || { echo "Failed to download $type platform $platform"; exit 1; }

    if ! file $repo_path | grep -q 'gzip compressed data'; then
        echo "Downloaded $type file platform $platform is not a valid gzip file"
        exit 1
    fi

    mkdir -p $target_dir || { echo "Failed to create $type target directory for platform $platform"; exit 1; }
    tar -xzvf $repo_path -C $target_dir --strip-components=1 || { echo "Failed to extract $type platform $platform"; exit 1; }
}

OCEAN_TARGET_DIR=%s
OCEAN_REPO_PATH=/tmp/ocean-repository.tar.gz
OCEAN_GITHUB_URL="https://github.com/f-rambo/ocean/releases/download/${OCEAN_VERSION}/${SYSTEM}-${ARCH}-ocean-${OCEAN_VERSION}.tar.gz"
download_and_extract $ARCH $OCEAN_NAME $OCEAN_GITHUB_URL $OCEAN_TARGET_DIR $OCEAN_REPO_PATH

# Loop through SHIP platforms
for platform in "${PLATFORMS[@]}"; do
    SHIP_GITHUB_URL="https://github.com/f-rambo/ship/releases/download/${SHIP_VERSION}/${SYSTEM}-${platform}-ship-${SHIP_VERSION}.tar.gz"
    SHIP_TARGET_DIR="%s/${platform}"
    SHIP_REPO_PATH="/tmp/ship-repository-${platform}.tar.gz"
    download_and_extract $platform $SHIP_NAME $SHIP_GITHUB_URL $SHIP_TARGET_DIR $SHIP_REPO_PATH
done


OCEAN_SYSTEMED_CONF="/etc/systemd/system/ocean.service"

cat <<EOF > $OCEAN_SYSTEMED_CONF
[Unit]
Description=Ocean Service
After=network.target

[Service]
User=root
ExecStart=$OCEAN_TARGET_DIR/bin/ocean -conf $OCEAN_TARGET_DIR/configs/config.yaml
Restart=on-failure
WorkingDirectory=$OCEAN_TARGET_DIR

[Install]
WantedBy=multi-user.target
EOF

ENV=%s
sed -i 's/^  env: .*/  env: $ENV/' $OCEAN_TARGET_DIR/configs/config.yaml

systemctl daemon-reload && systemctl enable ocean && systemctl start ocean && systemctl status ocean
`, oceanPath, shipPath, scriptEnv)
}

func getShipStartScript(shipPath string) string {
	return fmt.Sprintf(`#!/bin/bash

# ship server not network

SHIP_TARGET_DIR=%s

if [ -z "$SHIP_TARGET_DIR" ]; then
    echo "Usage: $0 <SHIP_TARGET_DIR>"
    exit 1
fi

SHIP_SYSTEMED_CONF="/etc/systemd/system/ship.service"

cat <<EOF > $SHIP_SYSTEMED_CONF
[Unit]
Description=Ship Service
After=network.target

[Service]
User=root
ExecStart=$SHIP_TARGET_DIR/bin/ship -conf $SHIP_TARGET_DIR/configs/config.yaml
Restart=on-failure
WorkingDirectory=$SHIP_TARGET_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload && systemctl enable ship && systemctl start ship && systemctl status ship
`, shipPath)
}

func getdownloadAndCopyScript() string {
	return `#!/bin/bash

# Parameters
DOWNLOAD_URL=$1
FILE_NAME=$2
SERVER_IP=$3
USER_NAME=$4
SERVER_FILE_PATH=$5
PORT=$6

# Check if all parameters are provided
if [ $# -ne 6 ]; then
  echo "Usage: $0 <download_url> <file_name> <server_ip> <user_name> <server_file_path> <port>"
  exit 1
fi

# Check if the file already exists on the remote server
if ssh -p "$PORT" "$USER_NAME@$SERVER_IP" "[ -f '$SERVER_FILE_PATH/$FILE_NAME' ]"; then
  echo "File $FILE_NAME already exists on the remote server. Exiting."
  exit 0
fi

# Check if the file already exists locally
if [ -f "$FILE_NAME" ]; then
  echo "File $FILE_NAME already exists locally. Skipping download."
else
  # Download the file using curl
  if ! curl -o "$FILE_NAME" "$DOWNLOAD_URL"; then
    echo "Failed to download file from $DOWNLOAD_URL"
    exit 1
  fi
fi

# Copy the file to the specified path on the server
if ! scp -P "$PORT" "$FILE_NAME" "$USER_NAME@$SERVER_IP:$SERVER_FILE_PATH"; then
  echo "Failed to copy $FILE_NAME to $USER_NAME@$SERVER_IP:$SERVER_FILE_PATH"
  exit 1
else
  echo "File $FILE_NAME to $USER_NAME@$SERVER_IP:$SERVER_FILE_PATH copied successfully."
fi`
}
