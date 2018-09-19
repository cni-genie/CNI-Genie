GO_PATH = $(GOPATH)
SRCFILES = $(wildcard *.go) 
DEPS = $(SRCFILES) \
       iptables/iptables.go \
       ipset/ipset.go 

# Ensure that the dist directory is always created
MAKE_SURE_DIST_EXIST := $(shell mkdir -p dist)

.PHONY: policy-controller clean binary image
default: binary 

policy-controller: binary image

binary: clean dist/genie-policy

image:
	docker build --no-cache -t genie-policy .
	docker tag genie-policy:latest quay.io/huawei-cni-genie/genie-policy-controller:latest
	docker rmi genie-policy:latest

clean:
	rm -rf dist

dist/genie-policy: $(DEPS)
	@GOPATH=$(GO_PATH) CGO_ENABLED=0 go build -v -i -o dist/genie-policy \
	-ldflags "-X main.VERSION=1.0 -s -w" $(SRCFILES)
