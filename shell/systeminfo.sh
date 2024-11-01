#!/bin/bash

uuid=$(sudo dmidecode -s system-uuid)

os=$(uname -s)

arch=$(uname -m)
case $arch in
aarch64)
      arch="arm64"
      ;;
x86_64)
      arch="amd64"
      ;;
esac

memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
memory_gb=$(((memory_kb + 1048575) / 1048576))

cpu_cores=$(nproc)

if command -v nvidia-smi &>/dev/null; then
      gpu_count=$(nvidia-smi --query-gpu=name --format=csv,noheader | wc -l)
      gpu_info=$(nvidia-smi --query-gpu=name --format=csv,noheader)
else
      gpu_info="No NVIDIA GPU found"
      gpu_count=0
fi

total_disk_bytes=0
while IFS= read -r line; do
      # Ignore the header line
      if [[ "$line" == "Size" ]]; then
            continue
      fi

      # Extract size and unit
      size=$(echo $line | awk '{print $1}')
      unit=${size: -1}
      num=${size%?}

      # Remove decimal part if it exists
      num=$(echo "$num" | cut -d'.' -f1)

      # Convert size to bytes
      case $unit in
      K)
            size_bytes=$((num * 1024))
            ;;
      M)
            size_bytes=$((num * 1024 * 1024))
            ;;
      G)
            size_bytes=$((num * 1024 * 1024 * 1024))
            ;;
      T)
            size_bytes=$((num * 1024 * 1024 * 1024 * 1024))
            ;;
      *)
            size_bytes=0
            ;;
      esac

      total_disk_bytes=$((total_disk_bytes + size_bytes))
done < <(df -h --output=size | tail -n +2)

total_disk_gb=$(((total_disk_bytes + 1073741823) / 1073741824))

inner_ip=$(hostname -I | awk '{print $1}')

json_output=$(
      cat <<EOF
{
  "id": "$uuid",
  "os": "$os",
  "arch": "$arch",
  "mem": "${memory_gb}",
  "cpu": "$cpu_cores",
  "gpu": "$gpu_count",
  "gpu_info": "$gpu_info",
  "disk": "${total_disk_gb}",
  "inner_ip": "$inner_ip"
}
EOF
)

echo "$json_output"
