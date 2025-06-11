#!/bin/bash
set -e

ARCH=$(uname -m)

masterNodeIp=$1
if [ -z "$masterNodeIp" ]; then
      echo "Master node IP is required"
      exit 1
fi

k3sShellPath=$2
if [ -z "$k3sShellPath" ]; then
      echo "K3s shell path is required"
      exit 1
fi

podCidr=$3
if [ -z "$podCidr" ]; then
      echo "Pod CIDR is required"
      exit 1
fi

serviceCidr=$4
if [ -z "$serviceCidr" ]; then
      echo "Service CIDR is required"
      exit 1
fi

k3sVersion=$5
if [ -z "$k3sVersion" ]; then
      k3sVersion="v1.33.1+k3s1"
fi

# server --config /etc/rancher/k3s/config.yaml
# --cluster-init (cluster) Initialize a new cluster using embedded Etcd (default: false) [$K3S_CLUSTER_INIT]
# --disable value [ --disable value ]  (components) Do not deploy packaged components and delete any deployed components (valid values: coredns, servicelb, traefik, local-storage, metrics-server, runtimes)

sudo INSTALL_K3S_EXEC="server --cluster-init --tls-san ${masterNodeIp} --cluster-cidr ${podCidr} --service-cidr ${serviceCidr} --disable traefik --disable servicelb --disable metrics-server --disable local-storage --disable-helm-controller --disable-kube-proxy --flannel-backend=none --disable-network-policy" INSTALL_K3S_VERSION="${k3sVersion}" bash $k3sShellPath

mkdir -p $HOME/.kube
sudo cp /etc/rancher/k3s/k3s.yaml $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
echo "K3s installed successfully"

# sudo INSTALL_K3S_EXEC="server --cluster-init --tls-san 192.168.90.149 --disable traefik --disable servicelb --disable metrics-server --disable local-storage --disable-helm-controller --disable-kube-proxy --flannel-backend=none --disable-network-policy" INSTALL_K3S_VERSION="v1.33.1+k3s1" bash /home/frambo/k3s.sh
