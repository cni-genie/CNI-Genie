# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:


GOPATH=/product/workspace
SRCFILES=cni-genie.go
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')

# Ensure that the dist directory is always created
MAKE_SURE_DIST_EXIST := $(shell mkdir -p dist)

.PHONY: all binary plugin
default: all
all: plugin
plugin: dist/genie

.PHONY: clean
clean:
	rm -rf dist

release: clean

# Build the genie cni plugin
dist/genie: $(SRCFILES)
	mkdir -p $(@D)
	@GOPATH=$(GOPATH) CGO_ENABLED=0 go build -v -i -o dist/genie \
	-ldflags "-X main.VERSION=1.0 -s -w" cni-genie.go
