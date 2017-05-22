# Getting started

### Prerequisite

* Linux box with
  * We tested on Ubuntu 14.04 & 16.04
* Docker installed
* Kubernetes cluster running with CNI enabled
  * One easy way to bring up a cluster is to use [kubeadm](https://kubernetes.io/docs/getting-started-guides/kubeadm/). We used
  ```
  $ kubeadm init --use-kubernetes-version=v1.5.8-beta.0 --pod-network-cidr=10.244.0.0/16
  ```
* One(or more) CNI plugin(s) installed, e.g., Canal, Weave, Calico
  * Use this [link](https://github.com/projectcalico/canal/tree/master/k8s-install) to install Canal
  * Use this [link](https://www.weave.works/docs/net/latest/kube-addon/) to install Weave
  * Use this [link](http://docs.projectcalico.org/v2.2/getting-started/kubernetes/installation/hosted/) to install Calico

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
