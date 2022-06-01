# Makefile to test the wost-go library
.DEFAULT_GOAL := help

.FORCE: 

test:  ## run tests after cleaning cache
	go clean -cache -testcache
	go test -race -failfast -p 1 -cover -v ./pkg/...

clean: ## Clean cache and imports
	go clean -cache -testcache
	go mod tidy

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
