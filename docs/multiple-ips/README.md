
## Use Case

* Multiple IP addresses can be injected into a single container making the container reachable across multiple networks
   * User-story: In a serverless platform the “Request Dispatcher” container that receives requests from customers of all different tenants needs to be able to pass the request to the right tenant. As a result, is should be reachable on the networks of all tenants
   * User-story: Many Telecom vendors are adopting container technology. For a router/firewall application to run in a container, it needs to have multiple interfaces
   
## How it works

* Step 1: same as Step 1 in [README.md](https://github.com/Huawei-PaaS/CNI-Genie/blob/master/docs/README.md) 
  
* Step 2:
  * User inputs his network(s) of choice in **pod annotations**
  
    ![image](multiple-ips-how-step2.png)

* Step 3: same as Step 3 in [README.md](https://github.com/Huawei-PaaS/CNI-Genie/blob/master/docs/README.md)

* Step 4: same as Step 4 in [README.md](https://github.com/Huawei-PaaS/CNI-Genie/blob/master/docs/README.md)

* Step 5: 
  * Genie calls the network(s) requested by the user and injects Multiple IPs, one per each request, into a single container
  * The container reachable across multiple networks

    ![image](multi-interface.png)
