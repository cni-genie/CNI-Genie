# Getting started

### Prerequisite

* Linux box with
  * We tested on Ubuntu 14.04 & 16.04
* Docker installed
* Kubernetes cluster running with CNI enabled
  * One easy way to bring up a cluster is to use [kubeadm](https://kubernetes.io/docs/getting-started-guides/kubeadm/): 
      * We tested on Kubernetes 1.5, 1.6, 1.7, 1.8
      
      Till 1.7 version:
      ```
      $ kubeadm init --use-kubernetes-version=v1.7.0 --pod-network-cidr=10.244.0.0/16
      ```

      Version 1.8 onwards:
      ```
      $ kubeadm init --pod-network-cidr=10.244.0.0/16
      ```

      Next steps:
      ```
      $ mkdir -p $HOME/.kube
      $ sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
      $ sudo chown $(id -u):$(id -g) $HOME/.kube/config
      ```
      * To schedule pods on the master, e.g. for a single-machine Kubernetes cluster,
      
      Till 1.7 version, run:
      ```
      $ kubectl taint nodes --all dedicated-
      ```

      Version 1.8 onwards, run:
      ```
      $ kubectl taint nodes --all node-role.kubernetes.io/master-
      ```

      
* One (or more) CNI plugin(s) installed, e.g., Calico, Weave, Flannel
  * Use this [link](https://docs.projectcalico.org/v3.2/getting-started/kubernetes) to install Calico       
  * Use this [link](https://www.weave.works/docs/net/latest/kube-addon/) to install Weave      
  * Use this [link](https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml) to install Flannel

### Installing genie

We install genie as a Docker Container on every node

Till Kubernetes 1.7 version: 
```
$ kubectl apply -f https://raw.githubusercontent.com/cni-genie/CNI-Genie/master/conf/1.5/genie.yaml
```

Kubernetes 1.8 version onwards:
```
$ kubectl apply -f https://raw.githubusercontent.com/cni-genie/CNI-Genie/master/releases/v3.0/genie.yaml
```

### Building, Testing, Making changes to source code

Refer to our [Developer's Guide](developer-guide.md) section.

### Genie Logs

For now Genie logs are stored in /var/log/syslog
To see the logs:
```
$ cat /dev/null > /var/log/syslog

$ tail -f /var/log/syslog | grep 'CNI'
```

### Troubleshooting

* Note: one a single node cluster, after your Kubernetes master is initialized successfully, make sure you are able to schedule pods on the master by running:
```
$ kubectl taint nodes --all node-role.kubernetes.io/master-
```
* Note: most plugins use differenet installation files for Kuberenetes 1.5, 1.6, 1.7 & 1.8. Make sure you use the right one!
