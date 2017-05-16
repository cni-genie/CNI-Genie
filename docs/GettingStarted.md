# Getting started

## Prerequisite

We assume you've the following environment ready:

* Linux box [We tested this on Ubuntu 14.04]
* Docker installed
* Kubernetes cluster with Canal network plugin [Easiest way is to use kubeadm for bringing up a cluster - https://kubernetes.io/docs/getting-started-guides/kubeadm/]

## Install options
You may follow either of the options below:
1. Install CNI-Genie as Docker container
2. Install from source

### Installing as Docker Container
This is easier way to setup CNI-Genie on your cluster. If you want to build from source then do not follow the below steps, jump to the next section.

* Kubernetes versions up to 1.5:
```
$ kubectl apply -f https://raw.githubusercontent.com/Huawei-PaaS/CNI-Genie/master/conf/1.5/genie.yaml
```

* Kubernetes versions up to 1.6:
```
coming soon...
```

### Installing from source
The easiest way to get genie plugin binary built is:

```
$ vi Makefile
```
Change the GOPATH appropriately (right now it's hard-coded to my machine settings, this will be changed soon). 
Save the Makefile before moving ahead

```
$ make all
```
This should create a "genie" binary in the dest/ folder.

```
$ cp dist/genie /opt/cni/bin/genie
$ systemctl restart kubelet
$ systemctl status kubelet
```

#### Test process

To run ginkgo tests for CNI-Genie run the following command:
```
$ make test
```

#### Configuring CNI-Genie plugin

As this experiment expects your cluster to have multi-network plugins running on the slave nodes, you need to follow the below steps:

##### Steps to reset kubeadm (optional step - you don't need to do this if you already have kubeadm running)
Here are the steps to reset kubeadm
```
$ kubeadm reset
$ weave reset
$ apt-get remove kubelet kubeadm kubectl kubernetes-cni
$ apt-get install -y kubelet kubeadm kubectl kubernetes-cni
$ kubeadm init
$ kubectl taint nodes --all dedicated-
```
##### Apply canal network plugin to kubeadm
```
$ kubectl apply -f canal.yaml
```
We used canal.yaml from 
https://raw.githubusercontent.com/projectcalico/canal/master/k8s-install/kubeadm/canal.yaml
NOT
https://github.com/projectcalico/canal/blob/master/k8s-install/canal.yaml

##### Install weave network plugin on same node
```
$ curl -L git.io/weave -o /usr/local/bin/weave
$ chmod a+x /usr/local/bin/weave
$ weave launch
$ eval "$(weave env)"
```

##### Test to run Canal's calico and Weave outside Kubernetes (optional step)
```
$ CONTAINER_ID=`docker run -itd --net=none nginx sleep 100000`

$ PID=`docker inspect -f '{{.State.Pid}}' ${CONTAINER_ID}`

$ CNI_COMMAND=ADD CNI_CONTAINERID=$CONTAINER_ID CNI_NETNS=/proc/$PID/ns/net CNI_IFNAME=eth0 CNI_PATH=/opt/cni/bin:/opt/calico/bin  /opt/cni/bin/calico < /etc/cni/net.d/10-calico.conf

$ CNI_COMMAND=ADD CNI_CONTAINERID=$CONTAINER_ID CNI_NETNS=/proc/$PID/ns/net CNI_IFNAME=eth0 CNI_PATH=/opt/cni/bin:/opt/weave-net/bin  /opt/cni/bin/weave-net < /etc/cni/net.d/10-weave.conf
```
## Genie Logs

For now Genie logs are stored in /var/log/syslog
To see the logs:
```
$ cat /dev/null > /var/log/syslog

$ tail -f /var/log/syslog | grep 'CNI'
```

That's all for now!
