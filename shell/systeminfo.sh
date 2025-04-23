#!/bin/bash

uuid=$(dmidecode -s system-uuid)

ARCH=$(uname -m)
case $ARCH in
aarch64)
      ARCH="arm64"
      ;;
x86_64)
      ARCH="amd64"
      ;;
*)
      log "Error: Unsupported architecture $ARCH. Supported architectures are: aarch64, x86_64"
      exit 1
      ;;
esac

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" != "linux" ]]; then
      log "Error: Unsupported OS $OS"
      exit 1
fi

memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
memory_gb=$(((memory_kb + 1048575) / 1048576))

cpu_cores=$(nproc)

if command -v nvidia-smi &>/dev/null; then
      gpu_count=$(nvidia-smi --query-gpu=name --format=csv,noheader | wc -l)
      gpu_info=$(nvidia-smi --query-gpu=name --format=csv,noheader)
else
      gpu_info=""
      gpu_count=0
fi

# 获取未挂载磁盘信息
unpartitioned_disks=()
total_disk_bytes=0
while IFS= read -r line; do
      # 跳过包含 "rom" 或 "loop" 的设备
      if [[ $line =~ rom|loop ]]; then
            continue
      fi

      # 获取设备名和大小
      name=$(echo "$line" | awk '{print $1}')
      size=$(echo "$line" | awk '{print $4}')
      type=$(echo "$line" | awk '{print $6}')

      # 只处理类型为 "disk" 且没有分区的设备
      if [[ "$type" == "disk" ]] && ! grep -q "^[[:space:]]*${name}p[0-9]" <(lsblk -l); then
            # 提取数字和单位
            size_num=${size%[KMGT]*}
            unit=${size##*[0-9.]}

            # 根据单位转换为字节
            case $unit in
            K)
                  size_bytes=$(echo "$size_num * 1024" | bc)
                  ;;
            M)
                  size_bytes=$(echo "$size_num * 1024 * 1024" | bc)
                  ;;
            G)
                  size_bytes=$(echo "$size_num * 1024 * 1024 * 1024" | bc)
                  ;;
            T)
                  size_bytes=$(echo "$size_num * 1024 * 1024 * 1024 * 1024" | bc)
                  ;;
            *)
                  size_bytes=0
                  ;;
            esac

            total_disk_bytes=$((total_disk_bytes + size_bytes))
            unpartitioned_disks+=("{\"device\":\"/dev/$name\",\"name\":\"$name\",\"size\":\"$size_num\"}")
      fi
done < <(lsblk -l | grep "disk")

# 将未分区磁盘信息转换为JSON数组
disk_json=$(
      IFS=,
      echo "[${unpartitioned_disks[*]}]"
)

total_disk_gb=$((total_disk_bytes / (1024 * 1024 * 1024)))

# 获取IP地址
ip=$(hostname -I 2>/dev/null | awk '{print $1}')

if [ -z "$ip" ]; then
      ip=$(ip addr show | grep 'inet ' | grep -v '127.0.0.1' | awk '{print $2}' | cut -d/ -f1 | head -n 1)
fi

if [ -z "$ip" ]; then
      ip=$(ifconfig | grep 'inet ' | grep -v '127.0.0.1' | awk '{print $2}' | head -n 1)
fi

if [ -z "$ip" ]; then
      ip=""
fi

json_output=$(
      cat <<EOF
{
  "id": "$uuid",
  "os": "$OS",
  "arch": "$ARCH",
  "mem": "${memory_gb}",
  "cpu": "$cpu_cores",
  "gpu": "$gpu_count",
  "gpu_info": "$gpu_info",
  "disk": "${total_disk_gb}",
  "unpartitioned_disks": ${disk_json},
  "ip": "$ip"
}
EOF
)

echo "$json_output"
