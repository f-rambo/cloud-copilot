#! bin/bash

source ./cmd/infra/variables.sh $1

installPackageFilename=$(basename "$kubesprayDownloadUrl")
version=$(echo "$installPackageFilename" | cut -d 'v' -f 2 | cut -d '.' -f 1-3)
filePath="$scriptPath/$installPackageFilename"


# 检查文件是否已经存在
if [ -f "$installPackageFilename" ]; then
      echo "文件已存在，无需下载。"
else
      # 下载文件
      echo "开始下载文件..."
      curl -LO $kubesprayDownloadUrl
      # 解压文件
      echo "开始解压文件..."
      tar -zxf $installPackageFilename
fi

# 开始安装kubespray
echo "开始安装kubespray..."
cd $kubesprayDir

if [ ! -f "requirements.txt" ]; then
    echo "requirements.txt文件不存在"
    exit 1
fi

echo "开始安装依赖..."
pip3 install -r requirements.txt --break-system-packages


# copy inventory文件
if [ ! -d "inventory/$clusterName" ]; then
    echo "inventory/$clusterName文件不存在"
    echo "开始copy inventory文件..."
    cp -rfp inventory/sample "inventory/$clusterName"
fi

ansibleConfigPath="ansible.cfg"

# 设置超时时间
echo "开始设置超时时间..."
if grep -q "^\[defaults\]" $ansibleConfigPath && grep -q "^timeout=" $ansibleConfigPath; then
    sed -i '/^\[defaults\]/,/^$/ s/^timeout=.*/timeout=240/' $ansibleConfigPath
else
    sed -i '/^\[defaults\]/a timeout=240' $ansibleConfigPath
fi

# 设置cluster名称
# sed -i "s/^cluster_name:.*/cluster_name: $clusterName/" inventory/$clusterName/group_vars/k8s_cluster/k8s-cluster.yml

# 设置kubernetes版本
# sed -i "s/^kube_version:.*/kube_version: $kubeVersion/" inventory/$clusterName/group_vars/k8s_cluster/k8s-cluster.yml

# 设置网络插件
# sed -i "s/^kube_network_plugin:.*/kube_network_plugin: $kubeNetworkPlugin/" inventory/$clusterName/group_vars/k8s_cluster/k8s-cluster.yml

# 设置pod子网
# sed -i "s/^kube_pods_subnet:.*/kube_pods_subnet: $kubePodsSubnet/" inventory/$clusterName/group_vars/k8s_cluster/k8s-cluster.yml

# 设置service地址
# sed -i "s/^kube_service_addresses:.*/kube_service_addresses: $kubeServiceAddress/" inventory/$clusterName/group_vars/k8s_cluster/k8s-cluster.yml

# 设置dashboard是否启用
# sed -i "s/^# dashboard_enabled: false/dashboard_enabled: true/" inventory/$clusterName/group_vars/k8s_cluster/addons.yml

# 设置ingress-nginx是否启用
# sed -i "s/^ingress_nginx_enabled:.*/ingress_nginx_enabled: $ingressNginxEnabled/" inventory/$clusterName/group_vars/k8s_cluster/addons.yml

# 设置ingress-nginx是否启用hostNetwork
# sed -i "s/^# ingress_nginx_host_network: false/ingress_nginx_host_network: true/" inventory/$clusterName/group_vars/k8s_cluster/addons.yml

# 设置helm是否启用
# sed -i "s/^helm_enabled:.*/helm_enabled: $helmEnabled/" inventory/$clusterName/group_vars/k8s_cluster/addons.yml

# 设置etcd是否启用peerClientAuth
# sed -i "s/^# etcd_peer_client_auth: true/etcd_peer_client_auth: true/" inventory/$clusterName/group_vars/etcd.yml

# inventory文件

INI_FILE="inventory/$clusterName/inventory.ini"
# 从 YAML 文件中提取数据
NODES=$(yq '.nodes' ../$clusterPathFilename)

# 初始化计数器
ETCD_COUNT=1

# 写入 [all] 分组
echo "[all]" > $INI_FILE
for NODE in $(echo $NODES | jq -c '.[]'); do
    NAME=$(echo $NODE | jq -r '.name')
    HOST=$(echo $NODE | jq -r '.host')
    USER=$(echo $NODE | jq -r '.user')
    PASSWORD=$(echo $NODE | jq -r '.password')
    SUDO_PASSWORD=$(echo $NODE | jq -r '.sudo_password')
    ROLE=$(echo $NODE | jq -r '.role[]')

    # 写入节点信息
    # 如果角色包含 master，则写入 etcd_member_name
    if [[ $ROLE == *"master"* ]]; then
        echo "$NAME ansible_host=$HOST ip=$HOST access_ip=$HOST ansible_user=$USER ansible_password=$PASSWORD ansible_become_password=$SUDO_PASSWORD etcd_member_name=etcd$ETCD_COUNT" >> $INI_FILE
        ETCD_COUNT=$((ETCD_COUNT+1))
    else
        echo "$NAME ansible_host=$HOST ip=$HOST access_ip=$HOST ansible_user=$USER ansible_password=$PASSWORD ansible_become_password=$SUDO_PASSWORD" >> $INI_FILE
    fi

done

# 添加空行
echo "" >> $INI_FILE

# 写入其他分组
echo "[kube_control_plane]" >> $INI_FILE
for NODE in $(echo $NODES | jq -c '.[] | select(.role[] == "master")'); do
    NAME=$(echo $NODE | jq -r '.name')
    echo "$NAME" >> $INI_FILE
done

echo "" >> $INI_FILE

echo "[etcd]" >> $INI_FILE
for NODE in $(echo $NODES | jq -c '.[] | select(.role[] == "master")'); do
    NAME=$(echo $NODE | jq -r '.name')
    echo "$NAME" >> $INI_FILE
done

echo "" >> $INI_FILE

echo "[kube_node]" >> $INI_FILE
for NODE in $(echo $NODES | jq -c '.[] | select(.role[] == "worker")'); do
    NAME=$(echo $NODE | jq -r '.name')
    echo "$NAME" >> $INI_FILE
done

echo "" >> $INI_FILE

echo "[calico_rr]" >> $INI_FILE

echo "" >> $INI_FILE
echo "[k8s_cluster:children]
kube_control_plane
kube_node
calico_rr" >> $INI_FILE

echo "" >> $INI_FILE
# 完成
echo "数据已写入到 ini 文件中"

echo "done"

