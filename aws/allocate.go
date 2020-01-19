package aws

import (
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// AllocationResult contains a net.IP / Interface pair
type AllocationResult struct {
	*net.IP
	Interface Interface
}

// AllocateClient offers IP allocation on interfaces
type AllocateClient interface {
	AllocateIPsOn(intf Interface, batchSize int64) ([]*AllocationResult, error)
	AllocateIPsFirstAvailableAtIndex(index int, batchSize int64) ([]*AllocationResult, error)
	AllocateIPsFirstAvailable(batchSize int64) ([]*AllocationResult, error)
	DeallocateIP(ipToRelease *net.IP) error
}

type allocateClient struct {
	aws    *awsclient
	subnet SubnetsClient
}

// AllocateIPsOn allocates IPs on a specific interface.
func (c *allocateClient) AllocateIPsOn(intf Interface, batchSize int64) ([]*AllocationResult, error) {
	var allocationResults []*AllocationResult
	client, err := c.aws.newEC2()
	if err != nil {
		return nil, err
	}
	request := ec2.AssignPrivateIpAddressesInput{
		NetworkInterfaceId: &intf.ID,
	}

	limits, err := c.aws.ENILimits()
	if err != nil {
		return nil, err
	}
	available := limits.IPv4 - int64(len(intf.IPv4s))

	// If there are fewer IPs left than the batch size, request all the remaining IPs
	// batch size 0 conventionally means "request the limit"
	if batchSize == 0 || available < batchSize {
		batchSize = available
	}

	request.SetSecondaryPrivateIpAddressCount(batchSize)

	_, err = client.AssignPrivateIpAddresses(&request)
	if err != nil {
		return nil, err
	}

	registry := &Registry{}
	for attempts := 10; attempts > 0; attempts-- {
		newIntf, err := c.aws.getInterface(intf.Mac)
		if err != nil {
			time.Sleep(1.0 * time.Second)
			continue
		}

		if len(newIntf.IPv4s) != len(intf.IPv4s) {
			// New address detected
			for _, newip := range newIntf.IPv4s {
				found := false
				for _, oldip := range intf.IPv4s {
					if newip.Equal(oldip) {
						found = true
					}
				}
				if !found {
					// only return IPs that haven't been previously registered
					if exists, err := registry.HasIP(newip); err == nil && !exists {
						// New IP. Timestamp the addition as a free IP.
						registry.TrackIP(newip)
						ipcopy := newip // Need to copy
						allocationResult := &AllocationResult{
							&ipcopy,
							newIntf,
						}
						allocationResults = append(allocationResults, allocationResult)
					}
				}
			}
			if len(allocationResults) > 0 {
				return allocationResults, nil
			}
		}
		time.Sleep(1.0 * time.Second)
	}

	return nil, fmt.Errorf("Can't locate new IP address from AWS")
}

// AllocateIPsFirstAvailableAtIndex allocates IP addresses, skipping any adapter < the given index
// Returns a reference to the interface the IPs were allocated on
func (c *allocateClient) AllocateIPsFirstAvailableAtIndex(index int, batchSize int64) ([]*AllocationResult, error) {
	interfaces, err := c.aws.GetInterfaces()
	if err != nil {
		return nil, err
	}
	limits, err := c.aws.ENILimits()
	if err != nil {
		return nil, err
	}

	var candidates []Interface
	for _, intf := range interfaces {
		if intf.Number < index {
			continue
		}
		if int64(len(intf.IPv4s)) < limits.IPv4 {
			candidates = append(candidates, intf)
		}
	}

	subnets, err := c.subnet.GetSubnetsForInstance()
	if err != nil {
		return nil, err
	}

	sort.Sort(SubnetsByAvailableAddressCount(subnets))
	for _, subnet := range subnets {
		if subnet.AvailableAddressCount <= 0 {
			continue
		}
		for _, intf := range candidates {
			if intf.SubnetID == subnet.ID {
				return c.AllocateIPsOn(intf, batchSize)
			}
		}
	}

	return nil, fmt.Errorf("Unable to allocate - no IPs available on any interfaces")
}

// AllocateIPsFirstAvailable allocates IP addresses on the first available IP address
// Returns a reference to the interface the IPs were allocated on
func (c *allocateClient) AllocateIPsFirstAvailable(batchSize int64) ([]*AllocationResult, error) {
	return c.AllocateIPsFirstAvailableAtIndex(0, batchSize)
}

// DeallocateIP releases an IP back to AWS
func (c *allocateClient) DeallocateIP(ipToRelease *net.IP) error {
	client, err := c.aws.newEC2()
	if err != nil {
		return err
	}
	interfaces, err := c.aws.GetInterfaces()
	if err != nil {
		return err
	}
	for _, intf := range interfaces {
		for _, ip := range intf.IPv4s {
			if ipToRelease.Equal(ip) {
				request := ec2.UnassignPrivateIpAddressesInput{}
				request.SetNetworkInterfaceId(intf.ID)
				strIP := ipToRelease.String()
				request.SetPrivateIpAddresses([]*string{&strIP})
				_, err = client.UnassignPrivateIpAddresses(&request)
				return err
			}
		}
	}

	return fmt.Errorf("IP not found - can't release")
}
