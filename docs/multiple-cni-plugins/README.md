## You can find here our [existing & future features covered in CNI-Genie](../CNIGenieFeatureSet.md)

# Feature 1: CNI-Genie "Multiple CNI Plugins"

## Motivation behind Multiple CNI Plugins

Right now Kubernetes Kubelet running on a slave node connects to at most one CNI plugin only i.e. either Canal or Romana or Weave.
This CNI-Genie feature enbales a pod, scheduled to run on a Node, to pickup over runtime any of the existing CNI plugins running on that particular node.

The current limitation and the reason why Kubernetes cannot do this is that when you are starting the kubelet, you are expected to pass cni-plugin details as a part of 'kubelet' process. In this case you have to pick only one of the existing CNI plugins and pass it as a flag to the kubelet. Now we feel that's in a way too restrictive! What if we want certain set of pods to use Canal networking and other set of pods to use weave networking? This is currently not possible in Kubernetes. For any multi-network support we need changes to be done to the Kubernetes, which leads to backward compatibility issues.

So, CNI-Genie "Multiple CNI Plugins" feature is designed to solve this problem without touching the Kubernetes code! 

## What CNI-Genie feature 1, "Multiple CNI Plugins", enables?

![image](what-cni-genie.png)

## How CNI-Genie feature 1 works?

* Step 1: 
  * We start Kubelet with **"genie"** as the CNI **"type"**. Note that for this to work we must have already placed **genie** binary under /opt/cni/bin as detailed in [getting started](../GettingStarted.md)
  * This is done by passing /etc/cni/net.d/genie.conf to kubelet

```json
{
    "name": "k8s-pod-network",
    "type": "genie",
    "etcd_endpoints": "http://10.96.232.136:6666",
    "log_level": "debug",
    "policy": {
      "type": "k8s",
       "k8s_api_root": "https://10.96.0.1:443",
       "k8s_auth_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJjYWxpY28tY25pLXBsdWdpbi10b2tlbi13Zzh3OSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJjYWxpY28tY25pLXBsdWdpbiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6ImJlZDY2NTE3LTFiZjItMTFlNy04YmU5LWZhMTYzZTRkZWM2NyIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDprdWJlLXN5c3RlbTpjYWxpY28tY25pLXBsdWdpbiJ9.GEAcibv-urfWRGTSK0gchlCB6mtCxbwnfgxgJYdEKRLDjo7Sjyekg5lWPJoMopzzPu8_-Tddd-yPZDJc44NCGRep7_ovjjJdlQvjhc0g1XA7NS8W0OMNHUJAzueyn4iuEwDHR7oNS_nwMqsfzgCsiIRkc7NkQDtKaBj8GOYTz9126zk37TqXylh7hMKlwDFkv9vCBcPv-nYU22UM67Ux6emAtf1g1Yw9i8EfOkbuqURir66jtcnwh3HLPSYMAEyADxYtYAxG9Ca-HhdXXsvnQxhd4P0h2ctgg0_NLTO6WRX47C3GNheLmq0tNttFXya0mHhcElSPQFZftzGw8ZvxTQ"
      },
    "kubernetes": {
      "kubeconfig": "/etc/cni/net.d/genie-kubeconfig"
    }
}
```
  
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

![image](how-step5.png)


### You can find here our [CNI-Genie Feature Set](docs/CNIGenieFeatureSet.md)

### [High-Level Design](../HLD.md)

