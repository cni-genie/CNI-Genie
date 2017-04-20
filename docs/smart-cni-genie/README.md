## You can find here our [existing & future features covered in CNI-Genie](../CNIGenieFeatureSet.md)

# Feature 3: CNI-Genie "Smart CNI Plugin Selection"

# Introduction

K8s User assets:
  - External User assets of each K8s User.
  - things private to the K8s app, like:
    - credentials for accessing other services (docker private repos

- Upon creating a pod, the user can manually select the logical network, or multiple logical networks, that the pod should be added to
-	Alternatively, the use can decide to include no logical network in pod yaml configuration. In this case, CNI-Genie smartly selects one of the available CNI plugins
-	For this purpose, CNI-Genie should maintain a list of KPIs for all available CNI plugins. Examples of such KPIs are
  - Network latency
  - Network bandwidth
  - End-to-end response time  
  - Percentage of IP addresses used, i.e., (# of IP addresses used)/(Total # of IP addresses)
  - Occupancy rate
  - A questionnaire filled out by the user to find use-case-optimized CNI plugin

# How it should work

In this case user leaves it to CNI-Genie to decide ideal logical network to be selected for a pod. The pod yaml looks like this:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-smart-pick
  labels:
    app: web
  annotations:
    cni: "genie"
spec:
  containers:
    - name: key-value-store
      image: nginx:latest
      imagePullPolicy: IfNotPresent
      ports:
        - containerPort: 6379
```

# High level design for selection based on "Network Bandwidth" usage
   
* Option 1: Measure bandwidth usage via [iperf3](https://iperf.fr/).

In this case, we run a pair of iperf3 client & server pods on every available plugin. The iperf3 client is used to measure the bandwidth usage for a given plugin. 
       
    ![image](iperf3-test.png)
       
* Option 2: Measure bandwidth usage via [fasthall perf_test](https://github.com/fasthall/container-network)

This tool helps monitor bandwidth usage of containers. In CNI-Genie for a given plugin the bandwidth usage of all of the containers using that plugin is measured.

Note: both Option 1 and 2 can be used to

 * Either periodically meaure and log bandwidth usage and CNI-Genie can **retreive the logs** when needed
 * Or to meaure bandwidth usage on-the-fly and CNI-Genie can **compare real-time measures** when needed
  
