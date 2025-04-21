PACKAGE_PATH := internal/biz

GO_FILES := $(wildcard $(PACKAGE_PATH)/*.go)
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*test*.go), $(GO_FILES))
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*biz*.go), $(GO_FILES))

GOPATH:=$(shell go env GOPATH)
VERSION=v0.0.1
SERVER_NAME=cloud-copilot
AUTHOR=frambo
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

.PHONY: generate
generate:
	make api
	make conf
	go mod tidy
	@cd cmd/${SERVER_NAME} && wire && cd -

.PHONY: run
run:
	go run ./cmd/${SERVER_NAME} -conf ./configs

.PHONY: build
build:
	mkdir -p bin/ && CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ./bin/$(SERVER_NAME) ./cmd/${SERVER_NAME}

platforms = linux/amd64 linux/arm64
.PHONY: build-all
build-all:
	@for platform in $(platforms); do \
		GOOS=$$(echo $$platform | cut -f1 -d/) && \
		GOARCH=$$(echo $$platform | cut -f2 -d/) && \
		IMAGE_NAME=$(IMG)-$$GOARCH && \
		docker build --platform $$platform -t $$IMAGE_NAME -f Dockerfile.dev . && \
		docker run -it -d --rm -v ./:/go/src/$(SERVER_NAME) --name $(SERVER_NAME)-$(VERSION)-$$GOARCH $$IMAGE_NAME && \
		docker exec -it $(SERVER_NAME)-$(VERSION)-$$GOARCH sh -c "cd $(SERVER_NAME) && CGO_ENABLED=1 GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags \"-s -w -X main.Version=$(VERSION)\" -o ./bin/$(SERVER_NAME)-$$GOARCH ./cmd/${SERVER_NAME}" && \
		docker rm -f $(SERVER_NAME)-$(VERSION)-$$GOARCH; \
	done

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) -f Dockerfile .

.PHONY: docker-run
docker-run:
	docker run -it -d --rm -p 8000:8000 -p 9000:9000 -v ./configs/:/data/conf --name $(SERVER_NAME)-$(VERSION) $(IMG) 

.PHONY: docker-dev
docker-dev:
	docker build -t $(IMG)-dev -f Dockerfile.dev .
	docker run -it -d --rm -v ./:/go/src/$(SERVER_NAME) --name $(SERVER_NAME)-dev $(IMG)-dev

.PHONY: package
package:
	[ -d "./resource" ] && tar -C ./ -czvf resource-$(VERSION).tar.gz ./resource || echo "resource directory not found"
	
	if [ -f "./bin/$(SERVER_NAME)-arm64" ]; then \
		mkdir -p ./$(SERVER_NAME)-arm64-$(VERSION) && \
		cp ./bin/$(SERVER_NAME)-arm64 ./$(SERVER_NAME)-arm64-$(VERSION)/$(SERVER_NAME) && \
		[ -d "./shell" ] && cp -r ./shell ./$(SERVER_NAME)-arm64-$(VERSION)/ && \
		[ -d "./component" ] && cp -r ./component ./$(SERVER_NAME)-arm64-$(VERSION)/ && \
		[ -d "./configs" ] && cp -r ./configs ./$(SERVER_NAME)-arm64-$(VERSION)/ && \
		tar -C./ -czvf $(SERVER_NAME)-arm64-$(VERSION).tar.gz ./$(SERVER_NAME)-arm64-$(VERSION) && \
		rm -rf ./$(SERVER_NAME)-arm64-$(VERSION); \
	fi
	
	if [ -f "./bin/$(SERVER_NAME)-amd64" ]; then \
		mkdir -p ./$(SERVER_NAME)-amd64-$(VERSION) && \
		cp ./bin/$(SERVER_NAME)-amd64 ./$(SERVER_NAME)-amd64-$(VERSION)/$(SERVER_NAME) && \
		[ -d "./shell" ] && cp -r ./shell ./$(SERVER_NAME)-amd64-$(VERSION)/ && \
		[ -d "./component" ] && cp -r ./component ./$(SERVER_NAME)-amd64-$(VERSION)/ && \
		[ -d "./configs" ] && cp -r ./configs ./$(SERVER_NAME)-amd64-$(VERSION)/ && \
		tar -C./ -czvf $(SERVER_NAME)-amd64-$(VERSION).tar.gz ./$(SERVER_NAME)-amd64-$(VERSION) && \
		rm -rf ./$(SERVER_NAME)-amd64-$(VERSION); \
	fi

.PHONY: docker-package
docker-package:
	make build-all && \
	make docker-dev && \
	docker exec -it $(SERVER_NAME)-dev sh -c "cd $(SERVER_NAME) && make package"

.PHONY: clean
clean:
	rm -rf ./resource-$(VERSION).tar.gz
	rm -rf ./$(SERVER_NAME)-arm64-$(VERSION).tar.gz
	rm -rf ./$(SERVER_NAME)-amd64-$(VERSION).tar.gz
	rm -rf ./bin

.PHONY: all
all:
	make api;
	make generate;
	make build-all;

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
