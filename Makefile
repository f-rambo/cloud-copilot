# 设置你的包路径和目标目录
PACKAGE_PATH := internal/biz
DESTINATION_DIR := mocks

# 获取所有的 .go 文件
GO_FILES := $(wildcard $(PACKAGE_PATH)/*.go)
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*test*.go), $(GO_FILES))
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*biz*.go), $(GO_FILES))

# 为每个 .go 文件生成对应的 mock 文件
MOCK_FILES := $(patsubst $(PACKAGE_PATH)/%.go,$(DESTINATION_DIR)/mock_%.go,$(GO_FILES))

# 设置 GOHOSTOS 和 GOPATH
GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=0.0.1
SERVER_NAME=ocean
AUTHOR=frambos
IMG=$(AUTHOR)/$(SERVER_NAME):$(VERSION)
# PLATFORMS = linux/amd64 linux/arm64 linux/riscv64 linux/ppc64le
PLATFORMS = linux/amd64 linux/arm64
BUILDER_NAME = $(SERVER_NAME)-multi-platform-buildx

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@v1.6.0

.PHONY: api
# generate api proto
api:
	protoc --proto_path=. \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
# build
build:
	mkdir -p bin/ && CGO_ENABLED=1 go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./cmd/ocean

.PHONY: generate
# generate
generate:
	go mod tidy
	@cd cmd/ocean && wire && cd -


.PHONY: docker-build
# docker-build
docker-build:
	docker build -t $(IMG) -f Dockerfile .

.PHONY: docker-run
# docker-run
docker-run:
	docker run -it -d --rm -p 8000:8000 -p 9000:9000 -v ./configs/:/data/conf --name $(SERVER_NAME)-$(VERSION) $(IMG) 

.PHONY: docker-stop
# docker-stop
docker-stop:
	docker stop $(SERVER_NAME)-$(VERSION)

.PHONY: docker-push
# docker-push
docker-push:
	docker push $(IMG)

.PHONY: run
# run
run:
	go run ./cmd/ocean -conf ./configs/

.PHONY: mock
mock: $(MOCK_FILES)
$(DESTINATION_DIR)/mock_%.go: $(PACKAGE_PATH)/%.go
	@mkdir -p $(DESTINATION_DIR)
	mockgen -source=$< -destination=$@ -package=mocks -write_package_comment

.PHONY: testcase
test:
	go test -v ./mocks/...

.PHONY: testserver
testserver:
	go test -v -count=1 -timeout=3h ./cmd/test/...

.PHONY: cleantestcache
cleantest:
	go clean -testcache

.PHONY: multi-platform-buildx
multi-platform-buildx:
	@if [ -z "$$(docker buildx inspect $(BUILDER_NAME) 2>/dev/null)" ]; then \
		echo "Creating and bootstrapping buildx builder: $(BUILDER_NAME)"; \
		docker buildx create --name $(BUILDER_NAME) --use; \
		docker buildx inspect $(BUILDER_NAME) --bootstrap; \
	else \
		echo "Using and bootstrapping existing buildx builder: $(BUILDER_NAME)"; \
		docker buildx use $(BUILDER_NAME); \
		docker buildx inspect $(BUILDER_NAME) --bootstrap; \
	fi

.PHONY: multi-platform-build
multi-platform-build:
	@for platform in $(PLATFORMS); do \
		image_name=$$platform/$(IMG); \
		echo "Building for platform $$platform to image $$image_name"; \
		docker buildx build -f Dockerfile.multi_platform_build --platform=$$platform -t $$image_name --load . ; \
	done

.PHONY: multi-platform-build-release
multi-platform-build-release:
	@mkdir -p ./built/
	@for platform in $(PLATFORMS); do \
		platform_formatted=$$(echo $$platform | tr '[:upper:]' '[:lower:]' | tr '/' '-'); \
		container_name=$$platform_formatted-$(SERVER_NAME)-$(VERSION); \
		echo "Building for platform $$platform_formatted to container $$container_name"; \
		docker run -it -d --rm --name $$container_name $$platform/$(IMG) ; \
		docker cp $$container_name:/app.tar.gz ./built/$$container_name.tar.gz ; \
		docker stop $$container_name; \
	done

.PHONY: multi-platform-build-push
multi-platform-build-push:
	@for platform in $(PLATFORMS); do \
		echo "Pushing for platform $$platform..."; \
		docker push $$platform/$(IMG) ; \
	done

.PHONY: multi-platform-build-clean
multi-platform-build-clean:
	@for platform in $(PLATFORMS); do \
		image_name=$$platform/$(IMG); \
		echo "Cleaning for platform $$platform to image $$image_name"; \
		docker rmi $$image_name ; \
		docker buildx rm $(BUILDER_NAME); \
	done

.PHONY: all
# generate all
all:
	make api;
	make generate;

# show help
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
