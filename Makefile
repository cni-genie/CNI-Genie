# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:


GO_PATH=$(GOPATH)
SRCFILES=cni-genie.go
TEST_SRCFILES=$(wildcard *_test.go)

# Ensure that the dist directory is always created
MAKE_SURE_DIST_EXIST := $(shell mkdir -p dist)

.PHONY: clean plugin policy-controller policy-controller-binary admission-controller admission-controller-binary test
default: plugin policy-controller-binary admission-controller-binary

plugin: clean dist/genie

test: dist/genie-test

clean:
	rm -rf dist

policy-controller: genie-policy
policy-controller-binary: genie-policy-binary
admission-controller: nw-admission-controller
admission-controller-binary: nw-admission-controller-binary

release: clean

# Build the genie cni plugin
dist/genie: $(SRCFILES)
	echo "Building genie plugin..."
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 go build -v -i -o dist/genie \
	-ldflags "-X main.VERSION=1.0 -s -w" cni-genie.go

nw-admission-controller-binary:
	cd controllers/network-admission-controller && make

nw-admission-controller:
	echo "Building genie network admission controller..."
	cd controllers/network-admission-controller && make admission-controller

genie-policy-binary:
	cd controllers/network-policy-controller && make

genie-policy:
	echo "Building genie network policy controller..."
	cd controllers/network-policy-controller && make policy-controller

# Build the genie cni plugin tests
dist/genie-test: $(TEST_SRCFILES)
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 ETCD_IP=127.0.0.1 PLUGIN=genie CNI_SPEC_VERSION=0.3.0 go test -args --testKubeVersion=$(testKubeVersion) --testKubeConfig=$(testKubeConfig)
