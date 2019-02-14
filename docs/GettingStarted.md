# Getting started

### Prerequisite

* Linux box with
  * We tested on Ubuntu 14.04 & 16.04
* Docker installed
* Kubernetes cluster running with CNI enabled
  * One easy way to bring up a cluster is to use [kubeadm](https://kubernetes.io/docs/getting-started-guides/kubeadm/): 
            
      Till 1.7 version:
      ```
      $ kubeadm init --use-kubernetes-version=v1.7.0 --pod-network-cidr=10.244.0.0/16
      ```

      1.8 version onwards:
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

      For 1.8 version onwards, run:
      ```
      $ kubectl taint nodes --all node-role.kubernetes.io/master-
      ```

      
* One (or more) CNI plugin(s) installed, e.g., Canal, Weave, Flannel
  * Use this [link](https://github.com/projectcalico/canal/tree/master/k8s-install) to install Canal       
  * Use this [link](https://www.weave.works/docs/net/latest/kube-addon/) to install Weave      
  * Use this [link](https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml) to install Flannel

### Installing genie components

We install genie as a Docker Container on every node

#### *Till Kubernetes 1.7 version:*
```
$ kubectl apply -f https://raw.githubusercontent.com/cni-genie/CNI-Genie/master/conf/1.5/genie.yaml
```

#### *Kubernetes 1.8 version onwards:*

CNI-Genie can be installed in the following two modes:

*Genie Complete (Installs genie with the support of multi networking as well as network policy implementation):*
```
$ kubectl apply -f https://raw.githubusercontent.com/cni-genie/CNI-Genie/master/conf/1.8/genie-complete.yaml
```

*Genie Plugin-only (Installs genie with multi networking support):*
```
$ kubectl apply -f https://raw.githubusercontent.com/cni-genie/CNI-Genie/master/conf/1.8/genie-plugin.yaml
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

* Note: on a single node cluster, after your Kubernetes master is initialized successfully, make sure you are able to schedule pods on the master by running:
```
$ kubectl taint nodes --all node-role.kubernetes.io/master-
```
* Note: most plugins use differenet installation files for different Kuberenetes versions. Make sure you use the right one!
