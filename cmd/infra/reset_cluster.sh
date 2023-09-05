#! bin/bash

source ./cmd/infra/variables.sh $1

cd $kubesprayDir

# 默认yes
sed -i -e 's/default: "no"/default: "yes"/g' playbooks/reset.yml

# 卸载集群
echo "开始卸载集群..."
ansible-playbook -i "inventory/$clusterName/inventory.ini" --become --become-user=root reset.yml

echo "done"