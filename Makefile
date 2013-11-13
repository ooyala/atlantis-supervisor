PROJECT_ROOT := $(shell pwd)
VENDOR_PATH  := $(PROJECT_ROOT)/vendor
LIB_PATH := $(PROJECT_ROOT)/lib
ATLANTIS_PATH := $(LIB_PATH)/atlantis

GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH)
export GOPATH

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

install:
	@echo "Installing Dependencies..."
	@rm -rf $(PROJECT_ROOT)/vendor
	@mkdir -p $(PROJECT_ROOT)/vendor || exit 2
	@GOPATH=$(PROJECT_ROOT)/vendor go get github.com/jigish/go-flags
	@GOPATH=$(PROJECT_ROOT)/vendor go get github.com/BurntSushi/toml
	@git clone ssh://git@github.com/ooyala/atlantis $(ATLANTIS_PATH)
	@GOPATH=$(PROJECT_ROOT)/vendor go get launchpad.net/gocheck
	@echo "Done."

test: clean copy-key
ifdef TEST_PACKAGE
	@echo "Testing $$TEST_PACKAGE..."
	@go test $$TEST_PACKAGE $$VERBOSE $$RACE
else
	@for p in `find ./src -type f -name "*.go" |sed 's-\./src/\(.*\)/.*-\1-' |sort -u`; do \
		echo "Testing $$p..."; \
		go test $$p || exit 1; \
	done
	@echo
	@echo "ok."
endif

fmt:
	@find src -name \*.go -exec gofmt -l -w {} \;
