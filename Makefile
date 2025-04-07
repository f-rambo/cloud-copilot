PACKAGE_PATH := internal/biz

GO_FILES := $(wildcard $(PACKAGE_PATH)/*.go)
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*test*.go), $(GO_FILES))
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*biz*.go), $(GO_FILES))

GOPATH:=$(shell go env GOPATH)
VERSION=v0.0.1
SERVER_NAME=cloud-copilot
AUTHOR=frambo9
IMG=$(AUTHOR)/$(SERVER_NAME):$(VERSION)

INTERNAL_BIZ_PBGO_FILES=$(shell find internal/biz -name *.pb.go)
API_PROTO_FILES=$(shell find api -name *.proto)

.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/favadi/protoc-go-inject-tag@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: api
api:
	protoc --proto_path=. \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: conf
conf:
	protoc --proto_path=. \
 	       --go_out=paths=source_relative:. \
	       internal/conf/conf.proto


.PHONY: build
build:
	mkdir -p bin/ && go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ./bin/$(SERVER_NAME) ./cmd/${SERVER_NAME}

.PHONY: generate
generate:
	go mod tidy
	@cd cmd/${SERVER_NAME} && wire && cd -


.PHONY: docker-build
docker-build:
	docker build -t $(IMG) -f Dockerfile .

.PHONY: docker-run
docker-run:
	docker run -it -d --rm -p 8000:8000 -p 9000:9000 -v ./configs/:/data/conf --name $(SERVER_NAME)-$(VERSION) $(IMG) 

.PHONY: docker-stop
docker-stop:
	docker stop $(SERVER_NAME)-$(VERSION)

.PHONY: docker-push
docker-push:
	docker push $(IMG)

.PHONY: docker-save
docker-save:
	docker save -o $(IMG).tar $(IMG)

.PHONY: run
run:
	go run ./cmd/${SERVER_NAME} -conf ./configs

.PHONY: all
all:
	make api;
	make generate;

help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
