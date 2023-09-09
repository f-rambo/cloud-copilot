FROM golang:1.19 AS builder

COPY . /app
WORKDIR /app

RUN apt-get update && apt-get install -y \
		ca-certificates netbase net-tools openssh-client && \
        rm -rf /var/lib/apt/lists/ && \
        apt-get autoremove -y && apt-get autoclean -y && \
        ssh-keygen -t rsa && \
        go env -w GOPROXY=https://goproxy.cn,direct && \
        make build

EXPOSE 8000
EXPOSE 9000

CMD ["./bin/server", "-conf", "configs"]