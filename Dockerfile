FROM golang:1.19 AS builder

COPY . /src
WORKDIR /src

RUN GOPROXY=https://goproxy.cn make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y \
		ca-certificates netbase openssh-client net-tools \
        sshpass expect curl yq jq gnupg \
        apt-transport-https python3 python3-pip && \
        curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /etc/apt/keyrings/kubernetes-archive-keyring.gpg && \
        curl -fsSL https://baltocdn.com/helm/signing.asc | gpg --dearmor -o  /etc/apt/keyrings/helm.gpg && \
        echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list && \
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | tee /etc/apt/sources.list.d/helm-stable-debian.list && \
        apt-get update && \
        apt-get install helm kubectl -y && \
        rm -rf /var/lib/apt/lists/ && \
        apt-get autoremove -y && apt-get autoclean -y


COPY --from=builder /src /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./bin/server", "-conf", "configs"]