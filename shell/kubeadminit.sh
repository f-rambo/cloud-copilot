#!/bin/bash
set -e

cluster_configuration="install/cluster-configuration.yaml"

kubeadm init --config "$cluster_configuration" --v=5

# 等待kubeadm生成配置文件

systemctl daemon-reload

systemctl enable kubelet

systemctl restart kubelet

kubectl config view

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# journalctl -u <service-name> -f
