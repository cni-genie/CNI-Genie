## You can find here our [existing & future features covered in CNI-Genie](../CNIGenieFeatureSet.md)

# Feature 2: CNI-Genie "Multiple IP Addresses"

## Use Case

* Multiple IP addresses can be injected into a single container making the container reachable across multiple networks
   * User-story: In a serverless platform the “Request Dispatcher” container that receives requests from customers of all different tenants needs to be able to pass the request to the right tenant. As a result, is should be reachable on the networks of all tenants
   * User-story: Many Telecom vendors are adopting container technology. For a router/firewall application to run in a container, it needs to have multiple interfaces
   
## Demo

[![asciicast](https://asciinema.org/a/120282.png)](https://asciinema.org/a/120282)
   
## How it should work

* Step 1: same as Step 1 in [README.md](../multiple-cni-plugins/README.md) 
  
* Step 2:
  * User inputs his network(s) of choice in **pod annotations**. For instance, the following yaml configurations can be used to get 2 IP addresses one from Weave and one from Canal:
  
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-multiips
  labels:
    app: web
  annotations:
    cni: "weave,canal"
spec:
  containers:
    - name: key-value-store
      image: nginx:latest
      imagePullPolicy: IfNotPresent
      ports:
        - containerPort: 6379
```

* Step 3: same as Step 3 in [README.md](../multiple-cni-plugins/README.md)

* Step 4: same as Step 4 in [README.md](../multiple-cni-plugins/README.md)

* Step 5: 
  * Genie calls the network(s) requested by the user and injects Multiple IPs, one per each request, into a single container
  * The container reachable across multiple networks

    ![image](multi-interface.png)

# Feature 2 Extension: CNI-Genie "Multiple IP Addresses PER POD"
   * This Work In-Progress (WIP) is an extension of Feature 2 where IP addresses are not only assigned to the container, but are also injected to the respective Pod object annotations. 
   * A [design document](https://docs.google.com/document/d/1zT2ofZzeowrJ-h4JWeKQyRGSDADJQssOoCFPpfwni7U/edit?usp=sharing) was prepared and shared with Kubernetes SIG Network community.
   * Watch the PoC demo to see how it works:
   
[![asciicast](https://asciinema.org/a/120338.png)](https://asciinema.org/a/120338)
   
