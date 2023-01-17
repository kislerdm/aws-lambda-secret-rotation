.DEFAULT_GOAL := help

help: ## Prints help message.
	@ grep -h -E '^[a-zA-Z0-9_-].+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1m%-30s\033[0m %s\n", $$1, $$2}'

tests: ## Run tests.
	@ go mod tidy && \
  		go test -timeout 3m --tags=unittest -v -coverprofile=.coverage.out . -coverpkg=. && \
		go tool cover -func .coverage.out && rm .coverage.out

compile:
	@ cd $(DIR) && \
 		test -d bin || mkdir -p bin && \
 		go mod tidy && \
  		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o bin/$(APP)-$(OS)-$(ARCH) -ldflags="-s -w" ./cmd/main.go

build: compile ## Builds the lambda binary and archives it.
