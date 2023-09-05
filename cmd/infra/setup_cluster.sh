#! bin/bash

source ./cmd/infra/variables.sh $1

cd $kubesprayDir

# 关闭防火墙
# echo "开始关闭防火墙..."
# ansible all -i "inventory/$clusterName/hosts.yaml" -m shell -a "sudo systemctl stop firewalld && sudo systemctl disable firewalld"

# 允许服务器之间允许IPv4转发
echo "开始允许服务器之间允许IPv4转发..."
ansible all -i "inventory/$clusterName/inventory.ini" -m shell -a "echo 'net.ipv4.ip_forward=1' | sudo tee -a /etc/sysctl.conf"

# 关闭swap
# echo "开始关闭swap..."
# ansible all -i "inventory/$clusterName/hosts.yaml" -m shell -a "sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab && sudo swapoff -a"

# 关闭selinux
# echo "开始关闭selinux..."
# ansible all -i "inventory/$clusterName/hosts.yaml" -m shell -a "sudo setenforce 0 && sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config"

# 关闭dnsmasq
# echo "开始关闭dnsmasq..."
# ansible all -i "inventory/$clusterName/hosts.yaml" -m shell -a "sudo systemctl stop dnsmasq && sudo systemctl disable dnsmasq"

# 关闭NetworkManager
# echo "开始关闭NetworkManager..."
# ansible all -i "inventory/$clusterName/hosts.yaml" -m shell -a "sudo systemctl stop NetworkManager && sudo systemctl disable NetworkManager"

# 开始安装kubernetes
echo "开始安装kubernetes..."
ansible-playbook -i "inventory/$clusterName/inventory.ini" --become --become-user=root cluster.yml

echo "done"
