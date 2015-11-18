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

ifndef DEB_VERSION
	DEB_VERSION := "0.1.0"
endif

ATLANTIS_PATH := $(LIB_PATH)/atlantis
BUILDER_PATH := $(LIB_PATH)/atlantis-builder
GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH):$(BUILDER_PATH)
export GOPATH

GOM := $(VENDOR_PATH)/bin/gom
GOM_VENDOR_NAME := vendor
export GOM_VENDOR_NAME

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go
	rm -f example/supervisor example/client example/monitor
	rm -rf $(VENDOR_PATH) $(LIB_PATH)

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

$(VENDOR_PATH):
	@echo "Installing Dependencies..."
	@mkdir -p $(VENDOR_PATH) || exit 2
	@GOPATH=$(VENDOR_PATH) go get github.com/ghao-ooyala/gom
	$(GOM) install
	@echo "Done."

init: clean $(VENDOR_PATH) copy-key
	@mkdir bin

build: init
	@go build -o bin/$(SERVER_BIN_NAME) example/supervisor.go
	@go build -o bin/$(CLIENT_BIN_NAME) example/client.go

deb: build
	@cp -a $(DEB) $(PKG)
	@mkdir -p $(DEB_INSTALL_DIR)
	@cp $(ATLANTIS_SECRET_DIR)/supervisor_master_id_rsa $(DEB_INSTALL_DIR)/master_id_rsa
	@chmod 600 $(DEB_INSTALL_DIR)/master_id_rsa
	@cp $(ATLANTIS_SECRET_DIR)/supervisor_master_id_rsa.pub $(DEB_INSTALL_DIR)/master_id_rsa.pub
	@cp -a bin $(DEB_INSTALL_DIR)/
	@mkdir -p $(DEB_CONFIG_DIR)
	@perl -p -i -e "s/__VERSION__/$(DEB_VERSION)/g" $(PKG)/$(PROJECT_NAME)/DEBIAN/control
	@cd $(PKG) && dpkg --build $(PROJECT_NAME) ../pkg

test: clean $(VENDOR_PATH) copy-key
ifdef TEST_PACKAGE
	@echo "Testing $$TEST_PACKAGE..."
	@go test $$TEST_PACKAGE $$VERBOSE $$RACE
else
	@for p in `find ./src -type f -name "*_test.go" |sed 's-\./src/\(.*\)/.*-\1-' |sort -u`; do \
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

lint:
	$(VENDOR_PATH)/bin/golint 
