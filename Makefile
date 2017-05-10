# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:


GO_PATH=$(GOPATH)
SRCFILES=cni-genie.go
TEST_SRCFILES=$(wildcard cni-genie_*_test.go)

# Ensure that the dist directory is always created
MAKE_SURE_DIST_EXIST := $(shell mkdir -p dist)

.PHONY: all plugin
default: clean all test
all: plugin
plugin: dist/genie

.PHONY: test
test: dist/genie-test

.PHONY: clean
clean:
	rm -rf dist

release: clean

# Build the genie cni plugin
dist/genie: $(SRCFILES)
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 go build -v -i -o dist/genie \
	-ldflags "-X main.VERSION=1.0 -s -w" cni-genie.go

# Build the genie cni plugin tests
dist/genie-test: $(TEST_SRCFILES)
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 ETCD_IP=127.0.0.1 PLUGIN=genie CNI_SPEC_VERSION=1.0 go test
