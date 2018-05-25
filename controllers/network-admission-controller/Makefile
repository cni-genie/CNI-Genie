GO_PATH = $(GOPATH)
SRCFILES = $(wildcard *.go)

# Ensure that the dist directory is always created
MAKE_SURE_DIST_EXIST := $(shell mkdir -p dist)

.PHONY: admission-controller clean binary image
default: binary

admission-controller: binary image

binary: clean dist/nw-adm-cont

image:
	docker build --no-cache -t genie-adm .
	docker tag genie-adm:latest quay.io/huawei-cni-genie/genie-admission-controller:latest
	docker rmi genie-adm:latest
clean:
	rm -rf dist

dist/nw-adm-cont: $(SRCFILES)
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 go build -v -i -o dist/network-admission-controller \
	-ldflags "-X main.VERSION=1.0 -s -w" $(SRCFILES)
