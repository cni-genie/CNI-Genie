#!/bin/sh

cd $GOPATH/src/github.com/cni-genie/CNI-Genie
# Start all the required plugins
bash -x plugins_install.sh -all

sleep 10

# E2E tests
make test-e2e testKubeVersion=1.7 testKubeConfig=/etc/kubernetes/admin.conf

sleep 20
# Delete all the installed plugins after usage
bash -x plugins_install.sh -deleteall

