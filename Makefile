# Copyright 2021 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

.DEFAULT_GOAL:=help

GOPATH  := $(shell go env GOPATH)
GOARCH  := $(shell go env GOARCH)
GOOS    := $(shell go env GOOS)
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE=on

TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)

## --------------------------------------
## Help
## --------------------------------------

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Tools
## --------------------------------------
.PHONY: tools
tools:
	mkdir -p $(TOOLS_BIN_DIR)
	hack/install-tools.sh

## --------------------------------------
## Lint
## --------------------------------------


.PHONY: go-lint
go-lint: tools
	hack/tools/bin/golangci-lint -v run

.PHONY: misspell
misspell: tools
	hack/check-misspell.sh

.PHONY: trailing-whitespace
trailing-whitespace: tools
	hack/check-trailing-whitespace.sh

.PHONY: eof-newline
eof-newline: tools
	hack/check-eof-newline.sh

.PHONY: language
language: tools
	hack/check-language.sh

.PHONY: lint
lint: tools go-lint misspell trailing-whitespace eof-newline language ## Lint codebase

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: ## Run go tests
	go test ./...



.PHONY: build
build: ## Compile and Build
	./hack/build.sh

## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin

.PHONY: clean-bin
clean-bin: ## Remove all downloaded tools
	rm -rf hack/tools/bin
