## Copyright 2014 Ooyala, Inc. All rights reserved.
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


PROJECT_NAME := $(shell pwd | xargs basename)
CLIENT_BIN_NAME := $(PROJECT_NAME)
SERVER_BIN_NAME := $(PROJECT_NAME)d


PKG := $(PROJECT_ROOT)/pkg
DEB := $(PROJECT_ROOT)/deb
DEB_INSTALL_DIR := $(PKG)/$(PROJECT_NAME)/opt/atlantis/supervisor
DEB_CONFIG_DIR := $(PKG)/$(PROJECT_NAME)/etc/atlantis/supervisor
INSTALL_DIR := /usr/local/bin/$(CLIENT_BIN_NAME)
CLIENT_CONFIG_DIR := /etc/atlantis/supervisor
CLIENT_CONFIG_NAME := client.default.toml
SERVER_CONFIG_NAME := server.default.toml
CLIENT_CONFIG_PATH := $(CLIENT_CONFIG_DIR)/$(CLIENT_CONFIG_NAME)
DISTRO := $(shell lsb_release -cs 2>/dev/null || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0 --match "[0-9]*\.[0-9]*\.[0-9]*")
DEB_REVISION := $(shell git log --pretty=oneline ^$(LAST_TAG) HEAD |wc |awk '{print $$1;}')
DEB_VERSION := $(LAST_TAG)-$(DEB_REVISION)



ATLANTIS_PATH := $(LIB_PATH)/atlantis
BUILDER_PATH := $(LIB_PATH)/atlantis-builder
GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH):$(BUILDER_PATH)
export GOPATH

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go
	rm -f example/supervisor example/client example/monitor

init: clean
	@mkdir bin

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go



build: init copy-key 
	@go build -o bin/$(SERVER_BIN_NAME) example/supervisor.go
	@go build -o bin/$(CLIENT_BIN_NAME) example/client.go

deb: build
	@cp -a $(DEB) $(PKG)
	@mkdir -p $(DEB_INSTALL_DIR)
##@cp $(ATLANTIS_SECRET_DIR)/supervisor_master_id_rsa $(DEB_INSTALL_DIR)/master_id_rsa
##@chmod 600 $(DEB_INSTALL_DIR)/master_id_rsa
##@cp $(ATLANTIS_SECRET_DIR)/supervisor_master_id_rsa.pub $(DEB_INSTALL_DIR)/master_id_rsa.pub
	@cp -a bin $(DEB_INSTALL_DIR)/
	@mkdir -p $(DEB_CONFIG_DIR)
	@perl -p -i -e "s/__VERSION__/$(DEB_VERSION)/g" $(PKG)/$(PROJECT_NAME)/DEBIAN/control
	@cd $(PKG) && dpkg --build $(PROJECT_NAME) ../pkg


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
