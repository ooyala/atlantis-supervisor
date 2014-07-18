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
ATLANTIS_PATH := $(LIB_PATH)/atlantis
BUILDER_PATH := $(LIB_PATH)/atlantis-builder
GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH):$(BUILDER_PATH)
export GOPATH

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go
	rm -f example/supervisor example/client example/monitor

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

$(ATLANTIS_PATH):
	@git clone ssh://git@github.com/ooyala/atlantis $(ATLANTIS_PATH)
	
$(VENDOR_PATH): | $(ATLANTIS_PATH)
	@echo "Installing Dependencies..."
	@rm -rf $(VENDOR_PATH)
	@mkdir -p $(VENDOR_PATH) || exit 2
	@GOPATH=$(VENDOR_PATH) go get github.com/jigish/go-flags
	@GOPATH=$(VENDOR_PATH) go get github.com/BurntSushi/toml
	@GOPATH=$(VENDOR_PATH) go get launchpad.net/gocheck
	@GOPATH=$(VENDOR_PATH) go get github.com/crowdmob/goamz/aws
	@GOPATH=$(VENDOR_PATH) go get github.com/crowdmob/goamz/s3
	@echo "Done."

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
