FROM golang:1.22 AS builder

COPY . /src
WORKDIR /src

RUN GOPROXY=https://goproxy.cn GOPRIVATE=github.com/f-rambo/ make build

FROM debian:stable-slim

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./bin/server", "-conf", "/data/conf"]