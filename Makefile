# Makefile to test the wost-go library
.DEFAULT_GOAL := help

.FORCE: 

test:  ## run tests
	go test -race -failfast -p 1 -v ./...


help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
