#! bin/bash

source ./cmd/infra/variables.sh $1

cd $kubesprayDir

# 检查SSH公钥是否存在
if [ -f ~/.ssh/id_rsa.pub ]; then
    echo "SSH公钥已经存在"
else
    # 如果公钥不存在，则生成新的公钥
    ssh-keygen -t rsa -b 4096 -C "$clusterName" -f ~/.ssh/id_rsa -N ""
fi


# 设置免密登录
echo "开始设置免密登录..."
ansible-playbook -i "inventory/$clusterName/inventory.ini" ../$scriptPath/servers.yaml