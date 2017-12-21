# cni-ipvlan-vpc-k8s: IPvlan Overlay-free Kubernetes Networking in AWS

`cni-ipvlan-vpc-k8s` contains a set of
[CNI](https://github.com/containernetworking/cni) and IPAM plugins to
provide a simple, host-local, low latency, high throughput, and [compliant
networking stack for
Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/networking/#kubernetes-model)
within [Amazon Virtual Private Coud
(VPC)](https://aws.amazon.com/vpc/) environments by making use of
[Amazon Elastic Network Interfaces
(ENI)](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html)
and binding AWS-managed IPs into Pods using the Linux kernel's IPvlan
driver in L2 mode.

The plugins are designed to be straightforward to configure and deploy
within a VPC. Kubelets boot and then self-configure and scale their IP
usage as needed, without requiring the often recommended complexities
of administering overlay networks, BGP, disabling source/destination
checks, or adjusting VPC route tables to provide per-instance subnets
to each host (which is limited to 50-100 entries per VPC). In short,
`cni-ipvlan-vpc-k8s` significantly reduces the network complexity
required to deploy Kubernetes at scale within AWS.

The maximum number of Pods per AWS instance is determined by [ENI
limits](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html#AvailableIpPerENI). Instance
types offering 8 ENIs can scale up to and beyond the default
Kubernetes limit of 110 pods per instance.

## Features

* Designed and tested on Kubernetes in AWS (v1.8 with CRI-O)
* No overlay network; very low overhead with IPvlan
* No external or local network services required outside of the AWS
  EC2 API; host-local scale up and scale down of network resources
* Unnumbered point-to-point interfaces connect Pods with their Kubelet
  and Daemon Sets using their well-known Kubernetes IPs and optionally
  provide IPv4 internet connectivity via NAT by directing traffic over
  the primary private IP of the boot ENI making use of Amazon's Public
  IPv4 addressing attribute feature.
* No asymmetric routing; no VPC routing table changes required
* Pod IPs are directly addressable from non-Kubernetes VPC
  hosts, easing migration of existing pre-Kubernetes service meshes
  and infrastructure.
* Automatic discovery of AWS resources, minimal plugin configuration
  required.

## How it Works

The primary EC2 boot ENI with its primary private IP is used as the IP
address for the node. Our CNI plugins manage additional ENIs and
private IPs on those ENIs to assign IP addresses to Pods.

Each Pod contains two network interfaces, a primary IPvlan interface
and an unnumbered point-to-point virtual ethernet interface. These
interfaces are created via a chained CNI execution.

![CNI Overview Diagram](./docs/cni.svg)

* IPvlan interface: The IPvlan interface with the Pod’s IP is used for
  all VPC traffic and provides minimal overhead for network packet
  processing within the Linux kernel. The master device is the ENI of
  the associated Pod IP. IPvlan is used in L2 mode with isolation
  provided from all other ENIs, including the boot ENI handling
  traffic for the Kubernetes control plane.
* Unnumbered point-to-point interface: A pair of virtual ethernet
  interfaces (veth) without IP addresses is used to interconnect the
  Pod’s network namespace to the default network namespace. The
  interface is used as the default route (non-VPC traffic) from the
  Pod and additional routes are created on each side to direct traffic
  between the node IP and the Pod IP over the link. For traffic sent
  over the interface, the Linux kernel borrows the IP address from the
  IPvlan interface for the Pod side and the boot ENI interface for the
  Kubelet side. Kubernetes Pods and nodes communicate using the same
  well-known addresses regardless of which interface (IPvlan or veth)
  is used for communication. This particular trick of “IP unnumbered
  configuration” is documented in
  [RFC5309](https://tools.ietf.org/html/rfc5309).

### Internet egress
For applications where Pods need to directly communicate with the
Internet, by setting the default route to the unnumbered
point-to-point interface, our stack can source NAT traffic from the
Pod over the primary private IP of the boot ENI, which enables making
use of Amazon’s Public IPv4 addressing attribute feature. When
enabled, Pods can egress to the Internet without needing to manage
Elastic IPs or NAT Gateways.

![CNI Overview Diagram](./docs/internet-egress.svg)


### Host namespace interconnect
Kubelets and Daemon Sets have high bandwidth, host-local access to all
Pods running on the instance — traffic doesn’t transit ENI
devices. Source and destination IPs are the well-known Kubernetes
addresses on either side of the connect.

* kube-proxy: We use kube-proxy in iptables mode and it functions as
  expected. The Pod's source IP is retained -- Kubernetes Services see
  connections from the Pod's source IP. The unnumbered point-to-point
  interface is used to loop traffic between kube-proxy in the default
  namespace for outbound connections created in the Pod namespace.
* [kube2iam](https://github.com/jtblin/kube2iam): Traffic from Pods to
  the AWS Metadata service transits over the unnumbered point-to-point
  interface to reach the default namespace before being redirected via
  destination NAT. The Pod’s source IP is maintained as kube2iam runs
  as a normal Daemon Set.

### VPC optimizations

Our design is heavily optimized for intra-VPC traffic where IPvlan is
the only overhead between the instance’s ethernet interface and the
Pod network namespace. We bias toward traffic remaining within the VPC
and not transiting the IPv4 Internet where veth and NAT overhead is
incurred. Unfortunately, many AWS services require transiting the
Internet; however, both DynamoDB and S3 offer VPC gateway endpoints.

While we have not yet implemented IPv6 support in our CNI stack, we
have plans to do so in the near future. IPv6 can make use of the
IPvlan interface for both VPC traffic as well as Internet traffic, due
to AWS’s use of public IPv6 addressing within VPCs and support for
egress-only Internet Gateways. NAT and veth overhead will not be
required for this traffic.

We’re planning to migrate to a VPC endpoint for DynamoDB and use
native IPv6 support for communication to S3. Biasing toward extremely
low overhead IPv6 traffic with higher overhead for IPv4 Internet
traffic is the right future direction.

# Using with Kubernetes

## Prerequisites

1. By default, we use a secondary (and tertiary, ...) ENI adapter for
   all Pod networking. This allows isolation by security groups or
   other constraints on the Kubelet control plane. This requires that
   the hosts you are running on can attach at least two ENI
   adapters. See:
   http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html
   Most hosts support > 1 adapter, except for some of the smallest
   hardware types.
1. AWS VPC with a minimum number of subnets equal to the maximum
   number of attached ENIs. In the normal case of supporting up to the
   default 110 Pods per instance, you'll want five subnets (one for
   the control plane on the boot ENI and four subnets for the Pod
   ENIs). The example configuration uses adapter index 1 onward for
   Pods. We recommend creating a secondary IPv4 CIDR block for
   Kubernetes deployments within existing VPCs and subnet
   appropriately for the number of ENIs.  In our primary region, we
   divide up our secondary IPv4 CIDR (/16) into 5 /20s per AZ with 3
   AZs.
1. (Optional) AWS subnets tagged if you want to limit which ones can
   be used.
1. The `kubelet` process _must_ be started with the `--node-ip` option
   if you also use `--cloud-provider=aws`. Use the primary IP on
   the boot ENI adapter (eth0).
1. AWS permissions allowing at least these actions on the _Kubelet_ role:

        "ec2:DescribeSubets"
        "ec2:AttachNetworkInterface"
        "ec2:AssignPrivateIpAddresses"
        "ec2:UnassignPrivateIpAddresses"
        "ec2:CreateNetworkInterface"
        "ec2:DescribeNetworkInterfaces"
        "ec2:DetachNetworkInterface"
        "ec2:DeleteNetworkInterface"
        "ec2:ModifyNetworkInterfaceAttribute"

    See [Security Considerations](#security-considerations) below for more on
    the implications of these permissions.


## Building

cni-ipvlan-vpc-k8s requires `dep` for dependency management. Please see
https://github.com/golang/dep#setup for build instructions. In a
pinch, you may `go get -u github.com/golang/dep/cmd/dep`.

    go get github.com/lyft/cni-ipvlan-vpc-k8s
    cd $GOPATH/src/github.com/lyft/cni-ipvlan-vpc-k8s
    make build

## Example Configuration

This example CNI conflist uses the bundled modified
`cni-ipvlan-vpc-k8s-ipvlan` plugin to create Pod IPs on the secondary
and above ENI adapters and chains with the
`cni-ipvlan-vpc-k8s-unnumbered-ptp` plugin to create unnumbered
point-to-point links back to the default namespace from each Pod. New
interfaces will be attached to subnets tagged with
`kubernetes_kubelet` = `true`, and created with the defined security
groups.

Routes are automatically formed for the VPC on the `ipvlan` adapter.

ipMasq is enabled to use the host-IP for egress to the Internet as
well as providing access to services such as `kube2iam`.

```
{
  "cniVersion": "0.3.1",
  "name": "cni-ipvlan-vpc-k8s",
  "plugins": [
  {
      "cniVersion": "0.3.1",
      "type": "cni-ipvlan-vpc-k8s-ipvlan",
      "mode": "l2",
      "master": "ipam",
      "ipam": {
          "type": "cni-ipvlan-vpc-k8s-ipam",
          "interfaceIndex": 1,
	      "subnetTags": {
	      "kubernetes_kubelet": "true"
	  },
	  "secGroupIds": [
	      "sg-1234",
	      "sg-5678"
	      ]
          }
    },
    {
        "cniVersion": "0.3.1",
        "type": "cni-ipvlan-vpc-k8s-unnumbered-ptp",
        "hostInterface": "eth0",
        "containerInterface": "veth0",
        "ipMasq": true
    }
    ]
}
```

## Security Considerations

In Kubernetes, pods and kubelets are assumed to have static IP addresses that
are assigned for the lifetime of the object. However, the EC2 IAM permissions
required by `cni-ipvlan-vpc-k8s` enable authorized principals to manipulate
network interfaces and IP addresses, which could be used to remap IP addresses
and "take over" the IP address of an existing pod or kubelet. Such an IP
address takeover could allow impersonation of a pod or kubelet at the network
layer, and disrupt the availability of your Kubernetes cluster.

IP address takeovers are possible in the following situations:
* Compromise of a kubelet instance configured to run `cni-ipvlan-vpc-k8s` with
  the required IAM permissions.
* Use (or abuse) of the EC2 ENI and IP Address manipulation APIs by a user or
  service in your AWS account authorized to do so.

Consider taking the following actions to reduce the likelihood and impact of IP
takeover attacks:
* Limit the number of principals authorized to manipulate ENIs and IP
  addresses.
* Do not rely exclusively on the Kubernetes control plane to ensure you're
  connected to the pod you expect. Deploy mutual TLS (mTLS) or other end-to-end
  authentication to authenticate clients and pods at the application layer.

# Get Support: Mailing Lists and Chat

 * Announcement list - new releases will be announced here:
   https://groups.google.com/forum/#!forum/cni-ipvlan-vpc-k8s-announce
 * Users discussion list:
   https://groups.google.com/forum/#!forum/cni-ipvlan-vpc-k8s-users
 * Gitter discussion: https://gitter.im/lyft/cni-ipvlan-vpc-k8s
