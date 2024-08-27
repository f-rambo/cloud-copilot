FROM golang:1.22 AS builder

COPY . /src
WORKDIR /src

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
ENV GOPRIVATE=github.com/f-rambo/

RUN make build

FROM debian:stable-slim

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./ocean", "-conf", "/data/conf"]