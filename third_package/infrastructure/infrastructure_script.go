package infrastructure

const (
	InstallScript = `
#!/bin/bash
# Springboard machine with lower performance, no docker installed, run directly on the machine

SYXTEM=$1
ARCH=$2
OCEAN_VERSION=$3
SHIP_VERSION=$4

install_tools() {
    if [ -f /etc/debian_version ]; then
        apt update
        apt install -y curl
        apt install -y tar
    elif [ -f /etc/redhat-release ]; then
        yum update -y
        yum install -y curl
        yum install -y tar
    else
        echo "unknown system type"
        exit 1
    fi
}

install_tools

OCEAN_GITHUB_URL="https://github.com/f-rambo/ocean/releases/download/${ARCH}/0.0.1/ocean.tar.gz"
SHIP_GITHUB_URL=""

OCEAN_TARGET_DIR="/app/ocean"
SHIP_TARGET_DIR="/app/ship"

OCEAN_REPO_PATH=/tmp/ocean-repository.tar.gz
SHIP_REPO_PATH=/tmp/ship-repository.tar.gz

curl -L $OCEAN_GITHUB_URL -o $OCEAN_REPO_PATH
curl -L $SHIP_GITHUB_URL -o $SHIP_REPO_PATH

mkdir -p $TARGET_DIR

tar -xzvf $OCEAN_REPO_PATH -C $OCEAN_TARGET_DIR --strip-components=1
tar -xzvf $SHIP_REPO_PATH -C $SHIP_TARGET_DIR --strip-components=1

cd $OCEAN_TARGET_DIR

./bin/ocean -conf ./configs/ -shell ./shell
`
	ShipShell = `#!/bin/bash
SHIP_DIR=$1

chmod +x $SHIP_DIR/bin/*

./$SHIP_DIR/bin/ship -conf ./configs/
`
)
