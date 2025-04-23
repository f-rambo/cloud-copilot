#!/bin/bash
set -e

log() {
      local message="$1"
      echo "$(date +'%Y-%m-%d %H:%M:%S') - $message"
}

ACTION=$1

if [ -z "$ACTION" ]; then
      log "Error: Action is required."
      exit 1
fi

if [ "$ACTION" != "get-ca-hash" ] && [ "$ACTION" != "get-token" ]; then
      log "Error: Action is invalid."
      exit 1
fi

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

function getCaHash() {
      openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | sha256sum | awk '{print $1}'
}

function getToken() {
      local token=""
      local token_output
      token_output=$(kubeadm token list 2>/dev/null)

      if [ $? -eq 0 ] && [ -n "$token_output" ]; then
            token=$(echo "$token_output" | awk 'NR>1 && $3 > "$(date +%Y-%m-%d)T$(date +%H:%M:%SZ)" {print $1}' | head -n1)
      fi

      if [ -z "$token" ]; then
            token=$(kubeadm token generate)
            if [ -z "$token" ]; then
                  log "Error: Failed to generate token"
                  return 1
            fi

            if ! kubeadm token create "$token" --ttl 24h; then
                  log "Error: Failed to create token"
                  return 1
            fi
      fi

      echo "$token"
}

case $ACTION in
get-ca-hash)
      getCaHash
      ;;
get-token)
      getToken
      ;;
esac
