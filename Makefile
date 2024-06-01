#### Dynamically Generated Interactive Menu ####

# Error Handling
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -c

# Name of this Makefile
MAKEFILE_NAME := $(lastword $(MAKEFILE_LIST))

# Special targets that should not be listed
EXCLUDE_LIST := menu all .PHONY

# Function to extract targets from the Makefile
define extract_targets
	$(shell awk -F: '/^[a-zA-Z0-9_-]+:/ {print $$1}' $(MAKEFILE_NAME) | grep -v -E '^($(EXCLUDE_LIST))$$')
endef

TARGETS := $(call extract_targets)

.PHONY: $(TARGETS) menu all

menu: ## Makefile Interactive Menu
	@# Check if fzf is installed
	@if command -v fzf >/dev/null 2>&1; then \
		echo "Using fzf for selection..."; \
		echo "$(TARGETS)" | tr ' ' '\n' | fzf > .selected_target; \
		target_choice=$$(cat .selected_target); \
	else \
		echo "fzf not found, using numbered menu:"; \
		echo "$(TARGETS)" | tr ' ' '\n' > .targets; \
		awk '{print NR " - " $$0}' .targets; \
		read -p "Enter choice: " choice; \
		target_choice=$$(awk 'NR == '$$choice' {print}' .targets); \
	fi; \
	if [ -n "$$target_choice" ]; then \
		$(MAKE) $$target_choice; \
	else \
		echo "Invalid choice"; \
	fi

# Default target
all: menu

help: ## This help function
	@egrep '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Targets (example targets listed below)
lint: ## Run lint
	gofmt -w .

test: ## Run test
	go test -v ./...

build: ## Run build
	go build .
	echo "Build completed, run ./gollama"

ci: ## build for linux and macOS
	mkdir -p ./dist/macos ./dist/linux_amd64 ./dist/linux_arm64
	GOOS=darwin GOARCH=arm64 go build . -o ./dist/macos/gollama
	GOOS=linux GOARCH=amd64 go build . -o ./dist/linux_amd64/gollama
	GOOS=linux GOARCH=arm64 go build . -o ./dist/linux_arm64/gollama
	echo "Build completed, run ./dist/macos/gollama or ./dist/linux_amd64/gollama or ./dist/linux_arm64/gollama"

run: ## Run
	go run *.go
