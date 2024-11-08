#!/bin/bash
set -e

kubectl drain --delete-local-data <node-name >--ignore-daemonsets

kubeadm reset

rm -rf ~/.kube

rm -rf /etc/kubernetes

rm -rf /etc/cni

# delete containerd images
for i in $(ctr -n k8s.io images list | awk '{print $3}' | grep -v REPOSITORY); do
      ctr -n k8s.io images remove "$i"
done
