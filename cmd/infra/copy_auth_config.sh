#! bin/bash

source ./cmd/infra/variables.sh $1

if [ -z "$2" ]
  then
    echo "Please provide the node name"
    exit 1
fi

nodeName=$2

cd $kubesprayDir

ansible $nodeName -i "inventory/$clusterName/inventory.ini" --become --become-user=root -m fetch -a "src=/root/.kube/config dest=/root/.kube/config flat=yes"

etcdRoleTaskPath="roles/etcd/defaults/main.yml"

etcdConfigDir=$(yq '.etcd_config_dir' $etcdRoleTaskPath | jq -r .)

certFile="$etcdConfigDir/ssl/member-$nodeName.pem"
keyFile="$etcdConfigDir/ssl/member-$nodeName-key.pem"
cacertFile="$etcdConfigDir/ssl/ca.pem"

echo "certFile: $certFile"
echo "keyFile: $keyFile"
echo "cacertFile: $cacertFile"

mkdir -p /app/etcd/ssl

ansible $nodeName -i "inventory/$clusterName/inventory.ini" --become --become-user=root -m fetch -a "src=$certFile dest=/app/etcd/ssl/member.pem flat=yes"
ansible $nodeName -i "inventory/$clusterName/inventory.ini" --become --become-user=root -m fetch -a "src=$keyFile dest=/app/etcd/ssl/member-key.pem flat=yes"
ansible $nodeName -i "inventory/$clusterName/inventory.ini" --become --become-user=root -m fetch -a "src=$cacertFile dest=/app/etcd/ssl/ca.pem flat=yes"

chmod 777 /app/etcd/ssl/*