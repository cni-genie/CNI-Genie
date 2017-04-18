# CNI Genie Version 1 High Level Design

CNI Genie v1 only supports features 1 and 3. For feature 3 only the case where a single type of logical network is specified by the user is supported in v1.

## Overview

From the viewpoint of Kubernetes kubelet CNI Genie is nothing but just another CNI plugin. As a result, no changes to Kubernetes are required. CNI Genie proxies for all of the existing CNI plugins, each providing a unique container networking solution, on the host.

![](overview.png)

## Location of configuration files

The configuration files describing the logical networks are still stored under the same default location, i.e.,

    --cni-conf-dir=/etc/cni/net.d/
    
For any existing CNI plugin, the user creates the corresponding directory subdirectory under

    /etc/cni/net.d/
e.g., 

    /etc/cni/net.d/canal/
    /etc/cni/net.d/calico/
    /etc/cni/net.d/flannel/

## How user inputs a logical network

In order to select their logical network of interest, the user makes use of the ‘ConfigMap’ object type. In the ConfigMap objects the name of the logical network configuration file is provided in form of a key, value pair. One example is given in the following:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: canal-config-file
  namespace: kube-system
data:
  cni_network_config: |-
    {
        "kubernetes": {
            "kubeconfig": "/etc/cni/net.d/canal/canal.conf"
        }
    }
```

The key, value pair is then used in the definition of the ‘Pod’ object, as the user creates his workload, to specify the logical network of choice for a given workload. An example of a how this can be done is depicted in the following:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: two-containers2
spec:
  restartPolicy: Never
  containers:
  - name: nginx-container
    image: nginx
    ports:
      - containerPort: 80
    env:
      - name: CNI_NETWORK_CONFIG
        valueFrom:
          configMapKeyRef:
            name: canal-config-file
            key: cni_network_config
```

## Detailed workflow

-	Once the pod is submitted to Kubernetes (Step 1), the scheduler selects a slave node to host the pod
-	At this point the kubelet on the host node gets triggered (Step 2) and 
-	sends a query to the CNI plugin, in this case CNI Genie, to get an IP address for the pod (Step 3)
-	The “Pod Name” is included in the body of the query message. CNI Genie uses the “Pod Name” to make a call to api-server to retrieve the name of the logical network selected by the user (Step 4)
-	CNI Genie uses the name of the logical network is retrieved to identify CNI plugin of choice and the network configuration file that should be passed to the CNI plugin (Step 5)
-	CNI Genie in turn sends a query to CNI plugin of choice using CNI messaging standard and gets back the IP address that should be used for the pod (Step 6)

A detailed illustration of the above described workflow is given in the following figure:

![](workflow.png)
