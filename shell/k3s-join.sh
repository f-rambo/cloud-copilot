#!/bin/bash
set -e

k3sUrl=$1
if [ -z "$k3sUrl" ]; then
      echo "K3s server URL is required"
      exit 1
fi

k3sToken=$2
if [ -z "$k3sToken" ]; then
      echo "K3s server token is required"
      exit 1
fi

k3sShellPath=$3
if [ -z "$k3sShellPath" ]; then
      echo "K3s shell path is required"
      exit 1
fi

k3sMaster=$4

K3S_URL=$k3sUrl

K3S_TOKEN=$k3sToken

bash $k3sShellPath

# curl -sfL https://get.k3s.io | K3S_URL=https://192.168.1.100:6443 K3S_TOKEN=K10b9e63... sh -

# curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --server https://192.168.1.100:6443" K3S_TOKEN=K10b9e63... sh -
