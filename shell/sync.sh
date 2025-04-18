#!/bin/bash
set -e

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

# Check minimum parameters
if [ "$#" -lt 4 ]; then
      log "Usage: $0 <local file path> <remote path> <remote username> <remote host1,host2... or host1 host2...>"
      exit 1
fi

LOCAL_FILE="$1"
REMOTE_PATH="$2"
REMOTE_USER="$3"
shift 3

# Handle both comma-separated and space-separated hosts
if [ "$#" -eq 1 ]; then
      # Comma-separated hosts
      IFS=',' read -ra HOSTS <<<"$1"
else
      # Space-separated hosts
      HOSTS=("$@")
fi

# Check if local path exists (file or directory)
if [ ! -e "$LOCAL_FILE" ]; then
      log "Error: Local path '$LOCAL_FILE' does not exist"
      exit 1
fi

# Execute file transfer for each host
for HOST in "${HOSTS[@]}"; do
      log "Starting transfer file '$LOCAL_FILE' to $REMOTE_USER@$HOST:$REMOTE_PATH"
      if scp -r -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$LOCAL_FILE" "$REMOTE_USER@$HOST:$REMOTE_PATH"; then
            log "File transfer successful to $HOST"
      else
            log "File transfer failed to $HOST"
            exit 1
      fi
done
