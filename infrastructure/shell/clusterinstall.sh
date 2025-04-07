#!/bin/bash
set -e

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

parse_yaml() {
      local yaml_string="$1"
      local query_path="$2"
      echo "$yaml_string" | yq eval "$query_path" -
}

cluster_data=$1
init_or_join=$2
is_control_plane=$3

if [ -z "$init_or_join" ]; then
      log "Error: Init or join is required."
      exit 1
fi

if [ "$init_or_join" != "init" ] && [ "$init_or_join" != "join" ]; then
      log "Error: Init or join is invalid."
      exit 1
fi

if [ -z "$cluster_data" ]; then
      log "Error: Cluster data is required."
      exit 1
fi

cluster_id=$(parse_yaml "$cluster_data" '.id')
if [ -z "$cluster_id" ]; then
      log "Error: Cluster id is required."
      exit 1
fi

cluster_name=$(parse_yaml "$cluster_data" '.name')
if [ -z "$cluster_name" ]; then
      log "Error: Cluster name is required."
      exit 1
fi

cluster_config=$(parse_yaml "$cluster_data" '.config')
if [ -z "$cluster_config" ]; then
      log "Error: Cluster config is required."
      exit 1
fi

cluster_version=$(parse_yaml "$cluster_data" '.kuberentes_version')
if [ -z "$cluster_version" ]; then
      log "Error: Cluster version is required."
      exit 1
fi

cilium_version=$(parse_yaml "$cluster_data" '.cilium_version')

function cluster_init() {

      log "Exec cluster init..."

      log "Start install ${cluster_name} cluster..."
      if [ -z "$cluster_config" ]; then
            log "Error: Cluster config is required."
            exit 1
      fi
      cluster_config_path=$HOME/cluster-config.yaml
      echo $cluster_config >$cluster_config_path
      if ! kubeadm init --config $cluster_config_path --v=5; then
            log "Error: Failed to init cluster."
            kubeadm reset --force
            exit 1
      fi
      log "Cluster init success."

      if ! kubeadm print join-command --token-ttl=24 >$HOME/kubeadm-join.sh; then
            log "Error: Failed to print join command."
            exit 1
      fi

      rm -f $HOME/.kube/config && mkdir -p $HOME/.kube && sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && sudo chown $(id -u):$(id -g) $HOME/.kube/config

      log "Install kubectl command..."

      arch=$(uname -m)
      if [ "$arch" == "x86_64" ]; then
            arch="amd64"
      fi
      if [ "$arch" == "aarch64" ]; then
            arch="arm64"
      fi

      if ! install -m 0755 $HOME/resource/${arch}/kubernetes/$cluster_version/kubectl /usr/local/bin/kubectl; then
            log "Error: Failed to install kubectl."
            exit 1
      fi

      log "Install kubectl command success."
}

function cluster_join() {
      local node_name=$1
      local node_ip=$2
      local node_user=$3

      log "Join node $node_name..."
      control_plane_command=""
      if [ ! -z "$is_control_plane" ]; then
            control_plane_command="--control-plane"
      fi
      if ! ssh -o StrictHostKeyChecking=no -i $HOME/.ssh/id_rsa $node_user@$node_ip "bash -s" $control_plane_command <$HOME/kubeadm-join.sh; then
            log "Error: Failed to join node $node_name."
            exit 1
      fi
      log "Join node $node_name success."
}

if [ "$init_or_join" == "init" ]; then
      cluster_init
      exist 0
fi

if [ "$init_or_join" == "join" ]; then
      nodes_length=$(echo "$cluster_data" | yq eval '.nodes | length' -)
      for ((i = 0; i < $nodes_length; i++)); do
            node_name=$(echo "$cluster_data" | yq eval ".nodes[$i].name" -)
            node_ip=$(echo "$cluster_data" | yq eval ".nodes[$i].ip" -)
            node_role=$(echo "$cluster_data" | yq eval ".nodes[$i].role" -) # 1 is master
            node_user=$(echo "$cluster_data" | yq eval ".nodes[$i].user" -)
            node_status=$(echo "$cluster_data" | yq eval ".nodes[$i].status" -) # 5 is ready
            if [ -z "$node_name" ] || [ -z "$node_ip" ] || [ -z "$node_user" ] [ -z "$node_role" ] || [ "$node_role" -ne 1 ] || [ "$node_role" -ne 5 ]; then
                  log "Skipping node $node_name is not a master or not ready."
                  continue
            fi
            cluster_join $node_name $node_ip $node_user
            exist 0
      done
fi
