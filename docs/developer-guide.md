## Developer's Guide

### Build process

After making any modification to source files, below steps can be followed to build and use the new binary/images. 

Note that you should install genie first before making changes to the source. This ensures genie conf file is generated successfully.

If changes to be made in admission controller and/or policy controller, then make sure to install the respective component
with container image pull policy as 'IfNotPresent'.

Please make sure to run the below commands with root privilege.

#### *Building and Using Genie plugin:*

Build genie binary by running:
```
make plugin
```
Place "genie" binary from dest/ into /opt/cni/bin/ directory.
```
cp dist/genie /opt/cni/bin/genie
```
#### *Building and Using network admission controller image:*

Admission controller image can be built by runnig:
```
make admission-controller
```
This will create a new image with the tag 'quay.io/huawei-cni-genie/genie-admission-controller:latest'.

Load this image with the same tag in the required node and then delete the genie-admission-controller pod runnig in that node. A new pod will automatically come up with the newly loaded image.

```
kubectl delete pod <genie-admission-controller pod name> -nkube-system
```

#### *Building and Using network policy controller image:*

Network policy controller image can be built by running:
```
make policy-controller
```
This will create a new image with the tag 'quay.io/huawei-cni-genie/genie-policy-controller:latest'.

Load this image with the same tag in the required node and then delete the genie-policy-controller pod runnig in that node. A new pod will automatically come up with the newly loaded image.

```
kubectl delete pod <genie-policy-controller pod name> -nkube-system
```

### Test process

#### *Unit Testing:*
To run unit test, execute below command:
```
make test
```

#### *E2E Testing:*

##### Prerequisites

A running kubernetes cluster is required to run the tests.

##### Running the tests

To run ginkgo tests for CNI-Genie run the following command:

If Kubernetes cluster is 1.7+
```
make test-e2e testKubeVersion=1.7 testKubeConfig=/etc/kubernetes/admin.conf
```

If Kubernetes cluster is 1.5.x
```
make test-e2e testKubeVersion=1.5

