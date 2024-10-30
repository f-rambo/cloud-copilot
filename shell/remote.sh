#!/bin/bash
set -e



REMOTE_USER=$1
REMOTE_IP=$2
REMOTE_PORT=$3
PRIVATE_KEY=$4

SHELL_SCRIPT_PATH=$5
if [[ ! -f "$SHELL_SCRIPT_PATH" ]]; then
  echo "Error: SHELL_SCRIPT_PATH must be a valid file."
  exit 1
fi
PARAM=$6

PRIVATE_KEY_FILE=$(mktemp)

echo "$PRIVATE_KEY" > $PRIVATE_KEY_FILE && chmod 600 $PRIVATE_KEY_FILE

trap 'rm -f "$PRIVATE_KEY_FILE"' EXIT

SHELL_SCRIPT_PATH=$(realpath "$SHELL_SCRIPT_PATH")
PRIVATE_KEY_FILE=$(realpath "$PRIVATE_KEY_FILE")

REMOTE_TEMP_DIR="/tmp"
REMOTE_SHELL_PATH="$REMOTE_TEMP_DIR/$(basename "$SHELL_SCRIPT_PATH")"

if ! scp -i "$PRIVATE_KEY_FILE" -P "$REMOTE_PORT" "$SHELL_SCRIPT_PATH" "$REMOTE_USER@$REMOTE_IP:$REMOTE_SHELL_PATH"; then
  echo "Error: SCP command failed."
  exit 1
fi

if ! ssh -i "$PRIVATE_KEY_FILE" -p "$REMOTE_PORT" "$REMOTE_USER@$REMOTE_IP" <<EOF
  bash $REMOTE_SHELL_PATH "$PARAM"
  EXIT_CODE=\$?
  if [[ \$EXIT_CODE -ne 0 ]]; then
    echo "Error: Remote script failed with exit code \$EXIT_CODE."
    exit \$EXIT_CODE
  fi
  rm -f $REMOTE_SHELL_PATH
  exit 0
EOF
then
  SSH_EXIT_CODE=$?
  echo "Error: SSH command failed with exit code $SSH_EXIT_CODE."
  exit $SSH_EXIT_CODE
fi