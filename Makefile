# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO           := go
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU        := $(FIRST_GOPATH)/bin/promu
pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX              ?= $(shell pwd)
BIN_DIR             ?= $(shell pwd)
DOCKER_IMAGE_NAME   ?= postgres-exporter
DOCKER_IMAGE_TAG    ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))


all: lint build test

lint:
	@echo ">> running golangci-lint"
	@golangci-lint run

test-short:
	@echo ">> running short tests"
	@$(GO) test -short -race $(pkgs)

test:
	@echo ">> running tests"
	@$(GO) test -race $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build: promu
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

tarball: promu
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

promu:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m))) \
		$(GO) install github.com/prometheus/promu@v0.13.0

.PHONY: all lint build test vet tarball docker promu
