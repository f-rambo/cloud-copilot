PACKAGE_PATH := internal/biz
DESTINATION_DIR := mocks

GO_FILES := $(wildcard $(PACKAGE_PATH)/*.go)
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*test*.go), $(GO_FILES))
GO_FILES := $(filter-out $(wildcard $(PACKAGE_PATH)/*biz*.go), $(GO_FILES))

MOCK_FILES := $(patsubst $(PACKAGE_PATH)/%.go,$(DESTINATION_DIR)/mock_%.go,$(GO_FILES))

GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=v0.0.1
SERVER_NAME=ocean
AUTHOR=frambos
IMG=$(AUTHOR)/$(SERVER_NAME):$(VERSION)
PLATFORMS = linux/arm64 linux/amd64

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
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@v1.6.0

.PHONY: api
api:
	protoc --proto_path=. \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./cmd/ocean

.PHONY: generate
generate:
	go mod tidy
	@cd cmd/ocean && wire && cd -


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

.PHONY: run
run:
	go run ./cmd/ocean -conf ./configs/

.PHONY: mock
mock: $(MOCK_FILES)
$(DESTINATION_DIR)/mock_%.go: $(PACKAGE_PATH)/%.go
	@mkdir -p $(DESTINATION_DIR)
	mockgen -source=$< -destination=$@ -package=mocks -write_package_comment

.PHONY: multi-platform-build
multi-platform-build:
	@if [ -z "$(PLATFORMS)" ] || [ -z "$(SERVER_NAME)" ] || [ -z "$(VERSION)" ]; then \
		echo "PLATFORMS, SERVER_NAME, or VERSION is not set"; \
		exit 1; \
	fi
	@mkdir -p ./built/
	@for platform in $(PLATFORMS); do \
		image_name=$$platform/$(IMG); \
		docker build --platform=$$platform -f Dockerfile.multi_platform_build -t $$image_name --load . ; \
		platform_formated=$$(echo $$platform | tr '[:upper:]' '[:lower:]' | tr '/' '-'); \
		container_name=$$platform_formated-$(SERVER_NAME)-$(VERSION); \
		docker run -it -d --rm --name $$container_name $$image_name; \
		docker cp $$container_name:/app.tar.gz ./built/$$container_name.tar.gz; \
		cd ./built && sha256sum $$container_name.tar.gz > $$container_name.tar.gz.sha256sum && cd - ; \
	done

.PHONY: download-resources
download-resources:
	@if [ -z "$(PLATFORMS)" ]; then \
		echo "PLATFORMS is not set"; \
		exit 1; \
	fi
	@mkdir -p ./resource/
	@for platform in $(PLATFORMS); do \
		image_name=resource/$$platform/$(IMG); \
		echo "Building image $$image_name"; \
		docker build --platform=$$platform -f Dockerfile.download_resource -t $$image_name --load . ; \
		platform_formated=$$(echo $$platform | tr '[:upper:]' '[:lower:]' | tr '/' '-'); \
		container_name=$$platform_formated-dind-container; \
		arch=$$(echo $$platform | cut -d '/' -f2); \
		mkdir -p ./resource/$$arch; \
		docker run --privileged --platform=$$platform --name $$container_name -d --rm -v ./resource/$$arch:/resource $$image_name; \
		docker exec $$container_name /bin/bash /shell/download.sh /resource; \
		docker stop $$container_name; \
		exit 0; \
	done

.PHONY: all
all:
	make api;
	make generate;
	make mock;

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
