## Developer's Guide

### Build process

After making any modification to source files, below steps can be followed to build and use the new binary. 

Note that you should install genie first before making changes to the source. This ensures genie conf file is generated successfully.

Please make sure to run the below commands with root privilege.

#### *Building and Using CNI-Genie plugin:*

Build genie binary by running:
```
make plugin
```
Place "genie" binary from dest/ into /opt/cni/bin/ directory.
```
cp dist/genie /opt/cni/bin/genie
```
### Test process

#### prerequisites

A running kubernetes cluster is required to run the tests.

#### Running the tests

To run ginkgo tests for CNI-Genie run the following command:

If Kubernetes cluster is 1.7+
```
make test testKubeVersion=1.7 testKubeConfig=/etc/kubernetes/admin.conf
```

If Kubernetes cluster is 1.5.x
```
make test testKubeVersion=1.5
