# CNI Genie: generic CNI network plugin

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
