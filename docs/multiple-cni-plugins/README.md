## You can find here our [existing & future features covered in CNI-Genie](../CNIGenieFeatureSet.md)

# Feature 1: CNI-Genie "Multiple CNI Plugins"

## Motivation behind Multiple CNI Plugins

Right now Kubernetes Kubelet running on a slave node connects to at most one CNI plugin only i.e. either Canal or Romana or Weave.
This CNI-Genie feature enbales a pod, scheduled to run on a Node, to pickup over runtime any of the existing CNI plugins running on that particular node.

The current limitation and the reason why Kubernetes cannot do this is that when you are starting the kubelet, you are expected to pass cni-plugin details as a part of 'kubelet' process. In this case you have to pick only one of the existing CNI plugins and pass it as a flag to the kubelet. Now we feel that's in a way too restrictive! What if we want certain set of pods to use Canal networking and other set of pods to use weave networking? This is currently not possible in Kubernetes. For any multi-network support we need changes to be done to the Kubernetes, which leads to backward compatibility issues.

So, CNI-Genie "Multiple CNI Plugins" feature is designed to solve this problem without touching the Kubernetes code! 

## What CNI-Genie feature 1, "Multiple CNI Plugins", enables?

![image](what-cni-genie.png)

## Demo

[![asciicast](https://asciinema.org/a/120279.png)](https://asciinema.org/a/120279)

## How CNI-Genie feature 1 works?

* Step 1: CNI-Genie should be installed as per instructions in [getting started](../GettingStarted.md)  
* Step 2:
  *  The user manually select the CNI plugin that he wants to add to a container upon creating a pod object. This goes under pod **annotations**
  
  *  Example 1: for Canal CNI plugin
  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: nginx-canal-master
    labels:
      app: web
    annotations:
      cni: "canal"
  spec:
    containers:
      - name: key-value-store
        image: nginx:latest
        imagePullPolicy: IfNotPresent
        ports:
          - containerPort: 6379
  ```

  *  Example 2: for Weave CNI plugin
  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: nginx-weave-master
    labels:
      app: web
    annotations:
      cni: "weave"
  spec:
    containers:
      - name: key-value-store
        image: nginx:latest
        imagePullPolicy: IfNotPresent
        ports:
          - containerPort: 6379
  ```

* Step 3
  * CNI-Genie gets pod name from args passed by kubelet
* Step 4
  * CNI-Genie gets pod annotations from api-server

![image](how-step3.png)

* Step 5
  * CNI-Genie calls the network choice requested by the user


### You can find here our [CNI-Genie Feature Set](../CNIGenieFeatureSet.md)

### [High-Level Design](../HLD.md)

