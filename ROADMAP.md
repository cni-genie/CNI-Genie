# CNI-Genie Roadmap

## Background & Motivation

CNI-Genie was originally designed to enable multihoming for Kubernetes pods by
enabling users to specify desired number of interfaces and the respective CNI
drivers for those interfaces.

While this offers great flexibility, users really only care that they get reliable multihoming capability with performant data-plane (traffic throughput) and control-plane (low network-ready latencies for pods) out of box. This became clear from a 2022 KubeCon EU stories. One particular talk discussed the complexities that were encountered using alternative solutions for multihoming. In the end, the other solution did not work because they designed their network for scale out rather than scale up. The ability to pick and choose CNI drivers becomes a much more appealing feature with a default that works well intituitively when networks start scaling.

## New Approach

For the past couple of years, we architected and built a pod networking solution
based on eBPF/XDP called the [Mizar project](https://github.com/centaurus-cloud/mizar).
Mizar was designed for fast data-plane performance by relying on eBPF/XDP to provide
the overlay networking that completely by-passes the host network stack to ferry
traffic between containers.

It was also built with a control-plane design to provide low-latency network-readiness for pods in order to handle the cloud native networking needs where pods rapidly come and go. Mizar also provides native multi-tenancy network isolation and was designed for scale out networking. The goal at that time was to provide a CNI networking solution for our scale out pod orchestration solution called project [Arktos](https://github.com/centaurus-cloud/arktos). We recently successfully integrated Mizar and Arktos and also demonstrated its multi-tenant networking capabilities in Arktos scaleout architecture at the Linux Foundation Open Source Summit in Austin, TX in June 2022.
 
We now realize that Mizar's eBPF/XDP technology can also address the critical cloud networking problems that we and others in the community face with multi-homed networking at scale.

## New Goals

We have identified following goals to integrate select Mizar's features into CNI-Genie:

- Add out-of-box fast & scalable eBPF/XDP based pod networking capability.
- Add ability for users to select the isolated networks to connect their pods into.
- Allow users to operate multiple groups of pods in their own isolated networks.
- Eliminate the (per-packet) overhead of network policies to achieve isolation.
- Add ability to CNI-Genie for users to select native network isolation using
  VPC isolation concept.
- Complete the control plane design to provide reliability and failover through
  distributed hash tables to store pod network groupings & connectivity information.
- Natively offer Network Quality of Service (QoS) to allow users to assign relative
  network traffic priorities to their pods.

## 2022 - 2023 Goals

For the next one year, we plan to take a few small steps and accomplish following:

- Identify and on-ramp new additional maintainer(s) for the project.
- Implement basic XDP multihomed pod networking features:
  - Implement pod-to-pod eBPF/XDP based multihomed networking with built-in isolation.
  - Implement service-to-pod eBPF/XDP based multihomed networking with built-in isolation.
- Implement simple and very basic XDP based egress gateway.
- Ensure ability to configure other CNI providers is retained.
- Restart community engagement for the project.
- Prototype and present new CNI-Genie roadmap features at conferences.
