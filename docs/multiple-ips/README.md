
## Use Case

* Multiple IP addresses can be injected into a single container making the container reachable across multiple networks
   * User-story: In a serverless platform the “Request Dispatcher” container that receives requests from customers of all different tenants needs to be able to pass the request to the right tenant. As a result, is should be reachable on the networks of all tenants
   * User-story: Many Telecom vendors are adopting container technology. For a router/firewall application to run in a container, it needs to have multiple interfaces
   
## How it should work

* Genie injects Multiple IPs into a single container
  * The container reachable across multiple networks

    ![image](multi-interface.png)
