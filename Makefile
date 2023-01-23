.DEFAULT_GOAL := help

help: ## Prints help message.
	@ grep -h -E '^[a-zA-Z0-9_-].+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1m%-30s\033[0m %s\n", $$1, $$2}'

tests: ## Run tests.
	@ go mod tidy && \
  		go test -timeout 3m --tags=unittest -v -coverprofile=.coverage.out . -coverpkg=. && \
		go tool cover -func .coverage.out && rm .coverage.out

PLUGIN := neon

test-plugin: ## Run plugin tests.
	@ cd plugin/$(PLUGIN) && go mod tidy && \
  		go test -timeout 3m --tags=unittest -v -coverprofile=.coverage.out . -coverpkg=. && \
		go tool cover -func .coverage.out && rm .coverage.out

compile:
	@ test -d bin || mkdir -p bin && \
 		cd plugin/$(PLUGIN) && \
 		go mod tidy && \
  		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../bin/$(PLUGIN)/lambda -ldflags="-s -w" ./cmd/lambda/main.go

PREFIX := $(notdir ${PWD})
TAG := ""

build: compile ## Builds the lambda binary and archives it.
	@ if [ $(TAG) = "" ]; then echo "specify TAG"; exit 137; fi
	@ cd bin/$(PLUGIN) && zip -9 $(PREFIX)_$(PLUGIN)_$(TAG).zip lambda && rm lambda
