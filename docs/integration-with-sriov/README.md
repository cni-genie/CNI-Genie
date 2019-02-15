## You can find here our [existing & future features covered in CNI-Genie](../CNIGenieFeatureSet.md)

# Steps to use sr-iov with CNI-Genie

1. Enable SR-IOV on the supported nodes (refer [Enable SR-IOV](https://github.com/cni-genie/CNI-Genie/blob/master/docs/integration-with-sriov/README.md#enable-sr-iov) section).

2. Build sr-iov binary. Build procedure can be followed from [here](https://github.com/hustcat/sriov-cni/blob/master/README.md).

3. Place the sriov binary in /opt/cni/bin/ directory in the sr-iov supported nodes of kubernetes cluster.

4. In the pod yaml, under annotation field, specify cni type as `sriov`.
Example pod yaml:
```
apiVersion: v1
kind: Pod
metadata:
  name: nginx-sriov
  labels:
    app: web
  annotations:
    cni: "sriov"
spec:
  containers:
    - name: key-value-store
      image: nginx:latest
      imagePullPolicy: IfNotPresent
      ports:
        - containerPort: 6379
```
5. CNI-Genie will automatically create a default conf file (/etc/cni/net.d/10-sriov.conf) on the node where the pod will be scheduled.
This conf file can also be placed manually and can be modified as per the requirement.

### Enable SR-IOV
Intel ixgbe NIC on Ubuntu(16.04), Debian or Linux Mint:
```
$ sudo vi /etc/modprobe.d/ixgbe.conf
options ixgbe max_vfs=8
```
Intel ixgbe NIC on CentOS, Fedora or RHEL:
```
$ sudo vi /etc/modprobe.conf
options ixgbe max_vfs=8
```
