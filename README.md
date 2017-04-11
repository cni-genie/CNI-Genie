# CNI-Genie

CNI-Genie enables container orchestrators (kubernetes, mesos) to seamlessly connect to choice of CNI plugins (calico, canal, romana, weave) configured on a Node. Without CNI-Genie, kubelet is bound to a signle CNI plugin passed to kubelet on start. CNI-Genie allows for multiple CNI plugins being available to kubelet simultaneously. 

[![Build Status](https://travis-ci.org/Huawei-PaaS/CNI-Genie.svg)](https://travis-ci.org/Huawei-PaaS/CNI-Genie)

***Note: this repo is still in inital development phase, so expect cracking sounds at times! :)***

***Also please note that this initial proto-type is tested only with Kubernetes build. Mesos will be coming soon...***

## What CNI-Genie does
This figure shows Kubernetes CNI Plugin landscape before and after CNI-Genie 

![image](what-cni-genie.png)

## Contributing
We always welcome contributions to our little experiment. 
Feel free to reach out to these folks:

karun.chennuri@huawei.com

kaveh.shafiee@huawei.com


# Why we created CNI-Genie?

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

### More docs here [Getting started](docs/GettingStarted.md), [README_v1.md](docs/README_v1.md), [Road map](docs/FutureEnhancements.md)
