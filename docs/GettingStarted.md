# Getting started

### Prerequisite

We assume you've the following environment ready:

* Linux box [We tested this on Ubuntu 14.04]
* Docker installed
* Kubernetes cluster with one (or more) CNI plugin(s), e.g., Canal, Weave, Calico
[Easiest way is to use kubeadm for bringing up a cluster - https://kubernetes.io/docs/getting-started-guides/kubeadm/]

### Installing genie

We install genie as a Docker Container on every node

* Kubernetes versions up to 1.5:
```
$ kubectl apply -f https://raw.githubusercontent.com/Huawei-PaaS/CNI-Genie/master/conf/1.5/genie.yaml
```
* Kubernetes versions up to 1.6:
```
coming soon...
```

### Making changes to and build from source

Note that you should install genie first before making changes to the source. This ensures genie conf file is generated successfully.

After making changes to source, build genie binary by running:
```
$ make all
```
Place "genie" binary from dest/ into /opt/cni/bin/ directory.
```
$ cp dist/genie /opt/cni/bin/genie
```

### Test process

To run ginkgo tests for CNI-Genie run the following command:
```
$ make test
```

### Genie Logs

For now Genie logs are stored in /var/log/syslog
To see the logs:
```
$ cat /dev/null > /var/log/syslog

$ tail -f /var/log/syslog | grep 'CNI'
```

That's all for now!
