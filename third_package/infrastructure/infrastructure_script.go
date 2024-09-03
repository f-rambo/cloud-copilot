package infrastructure

import "fmt"

var oceanPath string = "/app/ocean"
var shipPath string = "/app/ship"
var oceanDataTargzPackagePath string = "/tmp/oceandata.tar.gz"
var oceanDataTsha256sumFilePath string = "/tmp/oceandata.tar.gz.sha256sum"

var installScript string = fmt.Sprintf(`#!/bin/bash

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

OCEAN_DATA_TARGZ_PACKAGE_PATH=%s
OCEAN_DATA_TARGZ_PACKAGE_PATH_SHA256SUM=%s

sha256sum -c $OCEAN_DATA_TARGZ_PACKAGE_PATH_SHA256SUM || { echo "Failed to verify $OCEAN_DATA_TARGZ_PACKAGE_PATH_SHA256SUM"; exit 1; }

tar -xzvf $OCEAN_DATA_TARGZ_PACKAGE_PATH -C $HOME || { echo "Failed to extract $OCEAN_DATA_TARGZ_PACKAGE_PATH"; exit 1; }

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
        apt install -y curl tar file supervisor net-tools || { echo "Failed to install tools"; exit 1; }
    elif [ -f /etc/redhat-release ]; then
        yum update -y
        yum install -y curl tar file supervisor net-tools || { echo "Failed to install tools"; exit 1; }
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

# Create supervisor configuration for ocean
SUPERVISOR_CONF_DIR="/etc/supervisor/conf.d"
OCEAN_SUPERVISOR_CONF="$SUPERVISOR_CONF_DIR/ocean.conf"

cat <<EOF > $OCEAN_SUPERVISOR_CONF
[program:ocean]
command=$OCEAN_TARGET_DIR/bin/ocean -conf $OCEAN_TARGET_DIR/configs/
autostart=true
autorestart=true
stderr_logfile=/var/log/ocean.err.log
stdout_logfile=/var/log/ocean.out.log
EOF

# Start supervisord
supervisord -c /etc/supervisor/supervisord.conf

# Reload supervisor to apply the new configuration
supervisorctl reread
supervisorctl update
supervisorctl start ocean

# Query the status of all services managed by supervisor
supervisorctl status
`, oceanDataTargzPackagePath, oceanDataTsha256sumFilePath, oceanPath, shipPath)

var shipStartScript string = `#!/bin/bash

SHIP_TARGET_DIR=$1

if [ -z "$SHIP_TARGET_DIR" ]; then
    echo "Usage: $0 <SHIP_TARGET_DIR>"
    exit 1
fi

install_tools() {
    if [ -f /etc/debian_version ]; then
        apt update
        apt install -y supervisor net-tools || { echo "Failed to install tools"; exit 1; }
    elif [ -f /etc/redhat-release ]; then
        yum update -y
        yum install -y supervisor net-tools || { echo "Failed to install tools"; exit 1; }
    else
        echo "unknown system type"
        exit 1
    fi
}

install_tools

chmod +x $SHIP_TARGET_DIR/bin/ship

# Create supervisor configuration for ship
SUPERVISOR_CONF_DIR="/etc/supervisor/conf.d"
SHIP_SUPERVISOR_CONF="$SUPERVISOR_CONF_DIR/ship.conf"

cat <<EOF > $SHIP_SUPERVISOR_CONF
[program:ship]
command=$SHIP_TARGET_DIR/bin/ship -conf $SHIP_TARGET_DIR/configs/
autostart=true
autorestart=true
stderr_logfile=/var/log/ship.err.log
stdout_logfile=/var/log/ship.out.log
EOF

# Start supervisord
supervisord -c /etc/supervisor/supervisord.conf

# Reload supervisor to apply the new configuration
supervisorctl reread
supervisorctl update
supervisorctl start ship

# Query the status of all services managed by supervisor
supervisorctl status
`
