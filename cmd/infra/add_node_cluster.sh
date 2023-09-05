#! bin/bash

source ./cmd/infra/variables.sh $1

cd $kubesprayDir

# 添加节点
echo "开始添加节点..."
ansible-playbook -i "inventory/$clusterName/inventory.ini" --become --become-user=root scale.yml

echo "done"