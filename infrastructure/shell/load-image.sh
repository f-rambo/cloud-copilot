#!/bin/bash
set -e

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

# 检查必要的参数
if [ "$#" -lt 3 ]; then
      echo "Usage: $0 <image_pattern> <remote_host> <remote_user>"
      echo "Example: $0 'nginx*' remote.server.com username"
      exit 1
fi

IMAGE_PATTERN="$1"
REMOTE_HOST="$2"
REMOTE_USER="$3"
TEMP_DIR="/tmp/docker-images"

mkdir -p "$TEMP_DIR"

log "Finding images matching pattern: $IMAGE_PATTERN"
IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "$IMAGE_PATTERN")

if [ -z "$IMAGES" ]; then
      log "No images found matching pattern: $IMAGE_PATTERN"
      exit 1
fi

for IMAGE in $IMAGES; do
      log "Processing image: $IMAGE"
      FILENAME=$(echo "$IMAGE" | tr '/' '_' | tr ':' '_')

      log "Saving image to tar file..."
      docker save "$IMAGE" -o "$TEMP_DIR/${FILENAME}.tar"

      log "Copying image to remote server..."
      scp "$TEMP_DIR/${FILENAME}.tar" "${REMOTE_USER}@${REMOTE_HOST}:/tmp/"

      log "Importing image to containerd on remote server..."
      ssh "${REMOTE_USER}@${REMOTE_HOST}" "sudo ctr -n k8s.io images import /tmp/${FILENAME}.tar && rm /tmp/${FILENAME}.tar"

      rm "$TEMP_DIR/${FILENAME}.tar"
done

rmdir "$TEMP_DIR"

log "All images have been processed successfully"
