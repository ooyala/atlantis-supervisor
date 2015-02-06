##
## This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
## except in compliance with the License. You may obtain a copy of the License at
## http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software distributed under the License is
## distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and limitations under the License.

PROJECT_ROOT := $(shell pwd)
ifeq ($(shell pwd | xargs dirname | xargs basename),lib)
	LIB_PATH := $(shell pwd | xargs dirname)
	VENDOR_PATH := $(shell pwd | xargs dirname | xargs dirname)/vendor
else
	LIB_PATH := $(PROJECT_ROOT)/lib
	VENDOR_PATH := $(PROJECT_ROOT)/vendor
endif
ATLANTIS_PATH := $(LIB_PATH)/atlantis
BUILDER_PATH := $(LIB_PATH)/atlantis-builder

DEB_STAGING := $(PROJECT_ROOT)/staging
PKG_BIN_DIR := $(DEB_STAGING)/opt/atlantis-supervisor/bin
BIN_DIR := $(PROJECT_ROOT)/bin

ifndef VERSION
	VERSION := "0.1.0"
endif

GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH):$(BUILDER_PATH)
export GOPATH

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go
	rm -f example/supervisor example/client example/monitor
	@rm -rf $(DEB_STAGING) atlantis-supervisor_*.deb
	rm -rf lib/*

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

dependency-components:
	@git clone https://github.com/ooyala/atlantis $(ATLANTIS_PATH)
	@git clone https://github.com/ooyala/atlantis-builder $(BUILDER_PATH)

LAST_WORKING_SHA := "8fb2b14845d00ccc21d8847407d151383ba8ea2a"

install-deps: dependency-components
	@echo "Installing Dependencies..."
	@rm -rf $(VENDOR_PATH)
	@mkdir -p $(VENDOR_PATH) || exit 2
	@GOPATH=$(VENDOR_PATH) go get github.com/jigish/go-flags
	@GOPATH=$(VENDOR_PATH) go get github.com/BurntSushi/toml
	@GOPATH=$(VENDOR_PATH) go get launchpad.net/gocheck
	@GOPATH=$(VENDOR_PATH) go get github.com/crowdmob/goamz/aws
	@GOPATH=$(VENDOR_PATH) go get github.com/crowdmob/goamz/s3
	@GOPATH=$(VENDOR_PATH) go get github.com/fsouza/go-dockerclient
	@cd $(VENDOR_PATH)/src/github.com/fsouza/go-dockerclient; git reset --hard $(LAST_WORKING_SHA)
	@echo "Done."

build: install-deps example

deb: clean build
	@cp -a $(PROJECT_ROOT)/deb $(DEB_STAGING)
	@mkdir -p $(PKG_BIN_DIR) $(BIN_DIR)
	@cp example/client $(BIN_DIR)/supervisor-client
	@cp example/monitor $(PKG_BIN_DIR)
	@cp example/supervisor $(PKG_BIN_DIR)

	@sed -ri "s/__VERSION__/$(VERSION)/" $(DEB_STAGING)/DEBIAN/control
	@sed -ri "s/__PACKAGE__/atlantis-supervisor/" $(DEB_STAGING)/DEBIAN/control
	@dpkg -b $(DEB_STAGING) .

test: clean copy-key | $(VENDOR_PATH)
ifdef TEST_PACKAGE
	@echo "Testing $$TEST_PACKAGE..."
	@go test $$TEST_PACKAGE $$VERBOSE $$RACE
else
	@for p in `find ./src -type f -name "*.go" |sed 's-\./src/\(.*\)/.*-\1-' |sort -u`; do \
		[ "$$p" == 'atlantis/proxy' ] && continue; \
		echo "Testing $$p..."; \
		go test $$p || exit 1; \
	done
	@echo
	@echo "ok."
endif

.PHONY: example
example: copy-key
	@go build -o example/supervisor example/supervisor.go
	@go build -o example/client example/client.go
	@go build -o example/monitor example/monitor.go

fmt:
	@find src -name \*.go -exec gofmt -l -w {} \;
