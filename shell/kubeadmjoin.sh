#!/bin/bash
set -e

kubeadm join 192.168.1.100:6443 --config=join-control-plane.yaml --control-plane
