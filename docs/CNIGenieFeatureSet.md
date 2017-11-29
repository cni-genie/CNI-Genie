# Features covered in each CNI-Genie version:

## Existing features

**Feature 1: CNI-Genie "Multiple CNI Plugins"**
* Interface Connector to 3rd party CNI-Plugis. The user can [manually select one of the multiple CNI plugins](multiple-cni-plugins/README.md)

**Feature 2: CNI-Genie "Multiple IP Addresses"**
* Injects multiple IPs to a single container. The container is reachable using any of the [multiple IP Addresses](multiple-ips/README.md)

**Feature 3: CNI-Genie "Smart CNI Plugin Selection"**
* Intelligence in selecting the CNI plugin. CNI-Genie [watches the KPI of interest and selects](smart-cni-genie/README.md) the CNI plugin, accordingly

## Future features

**Feature 4: CNI-Genie "Network Isolation"**
* Dedicated 'physical' network for a tenant
* Isolated 'logical' networks for different tenants on a shared 'physical'network 

**Feature 5: CNI-Genie "Network Policy Engine"**
* [CNI-Genie network policy engine](network-policy/README.md) allows for network level ACLs 

**Feature 6: CNI-Genie "Real-time Network Switching"**
* Price minimization: dynamically switching workload to a cheaper network as network prices change
* Maximizing network utilization: dynamically switching workload to the less congested network at a threshold
