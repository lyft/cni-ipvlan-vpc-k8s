package freeip

import (
	"github.com/lyft/cni-ipvlan-vpc-k8s/aws"
	"github.com/lyft/cni-ipvlan-vpc-k8s/nl"
)

// FindFreeIPsAtIndex locates free IP addresses by comparing the assigned list
// from the EC2 metadata service and the currently used addresses
// within netlink. This is inherently somewhat racey - for example
// newly provisioned addresses may not show up immediately in metadata
// and are subject to a few seconds of delay.
func FindFreeIPsAtIndex(index int) ([]*aws.AllocationResult, error) {
	freeIps := []*aws.AllocationResult{}

	interfaces, err := aws.DefaultClient.GetInterfaces()
	if err != nil {
		return nil, err
	}
	assigned, err := nl.GetIPs()
	if err != nil {
		return nil, err
	}

	for _, intf := range interfaces {
		if intf.Number < index {
			continue
		}
		for _, intfIP := range intf.IPv4s {
			found := false
			for _, assignedIP := range assigned {
				if assignedIP.IPNet.IP.Equal(intfIP) {
					found = true
					break
				}
			}
			if !found {
				intfIPCopy := intfIP
				// No match, record as free
				freeIps = append(freeIps, &aws.AllocationResult{
					&intfIPCopy,
					intf,
				})
			}
		}
	}
	return freeIps, nil
}
