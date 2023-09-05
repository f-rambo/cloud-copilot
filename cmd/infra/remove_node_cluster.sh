#! bin/bash

source ./cmd/infra/variables.sh $1

cd $kubesprayDir

if [ -z "$2" ]
  then
    echo "Please provide the nodename"
    exit 1
fi

nodeNames=$2

# 删除节点
echo "开始删除节点..."
ansible-playbook -i "inventory/$clusterName/inventory.ini" --become --become-user=root remove-node.yml --extra-vars "node=$nodeNames"

echo "done"