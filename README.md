CNI-Genie enables orchestrators (kubernetes, mesos) for seamless connectivity to choice of CNI plugins (calico, canal, romana, weave) configured on a Node.

[![Build Status](https://travis-ci.org/Huawei-PaaS/CNI-Genie.svg)](https://travis-ci.org/Huawei-PaaS/CNI-Genie)

***Note: this repo is still in inital development phase, so expect cracking sounds at times! :)***

***Also please note that this initial proto-type is tested only with Kubernetes build. Mesos will be coming soon...***

# Getting started

## Prerequisite

We assume you've the following environment ready:

* Linux box [We tested this on Ubuntu 14.04]
* Docker installed
* Kubernetes cluster with Canal network plugin [Easiest way is to use kubeadm for bringing up a cluster - https://kubernetes.io/docs/getting-started-guides/kubeadm/]

## Build process
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

## Configuring CNI-Genie plugin

As this experiment expects your cluster to have multi-network plugins running on the slave nodes, you need to follow the below steps:

### Steps to reset kubeadm (optional step - you don't need to do this if you already have kubeadm running)
Here are the steps to reset kubeadm
```
$ kubeadm reset
$ weave reset
$ apt-get remove kubelet kubeadm kubectl kubernetes-cni
$ apt-get install -y kubelet kubeadm kubectl kubernetes-cni
$ kubeadm init
$ kubectl taint nodes --all dedicated-
```
### Apply canal network plugin to kubeadm
```
$ kubectl apply -f canal.yaml
```
We used canal.yaml from 
https://raw.githubusercontent.com/projectcalico/canal/master/k8s-install/kubeadm/canal.yaml
NOT
https://github.com/projectcalico/canal/blob/master/k8s-install/canal.yaml

### Install weave network plugin on same node
```
$ curl -L git.io/weave -o /usr/local/bin/weave
$ chmod a+x /usr/local/bin/weave
$ weave launch
$ eval "$(weave env)"
```

### Test to run Canal's calico and Weave outside Kubernetes (optional step)
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

$ tail -f /var/log/syslog | grep 'Calico CNI'
```

That's all for now!

## Contributing
We always welcome contributions to our little experiment. 
Feel free to reach out to these folks:

karun.chennuri@huawei.com

kaveh.shafiee@huawei.com


# Why we developed CNI-Genie?

CNI Genie is an add-on to [Kuberenets](https://github.com/kubernetes/kubernetes) open-source project and is designed to provide the following features:

1. Multiple CNI plugins are available to users in runtime. The user can offer any of the available CNI plugins to containers upon creating them
    - User-story: based on ‘performance’ requirements, ‘application’ requirements, “workload placement” requirements, the user could be interested to use different CNI plugins for different application groups
    - Different CNI plugins are different in terms of need for port-mapping, NAT, tunneling, interrupting host ports/interfaces

2. Multiple IP addresses can be injected into a single container making the container reachable across multiple networks
    - User-story: in a serverless platform the “Request Dispatcher” container that receives requests from customers of all different tenants needs to be able to pass the request to the right tenant. As a result, is should be reachable on the networks of all tenants
    - User-story: many Telecom vendors are adopting container technology. For a router/firewall application to run in a container, it needs to have multiple interfaces

3. Upon creating a pod, the user can manually select the logical network, or multiple logical networks, that the pod should be added to

4. If upon creating a pod no logical network is included in the yaml configuration, CNI Genie will automatically select one of the available CNI plugins
    - CNI Genie maintains a list of KPIs for all available CNI plugins. Examples of such KPIs are occupancy rate, number of subnets, response times

5. CNI Genie stores records of requests made to each CNI plugin for logging and auditing purposes and it can generate reports upon request

6. Network policy

7. Network access control

Note: CNI Genie is NOT a routing solution! It gets IP addresses from various CNSs

### More docs here [docs/README.md](docs/README.md), [Road map](docs/FutureEnhancements.md)
