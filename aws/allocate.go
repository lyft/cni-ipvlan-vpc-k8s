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
	AllocateIPOn(intf Interface) (*AllocationResult, error)
	AllocateIPFirstAvailableAtIndex(index int) (*AllocationResult, error)
	AllocateIPFirstAvailable() (*AllocationResult, error)
	DeallocateIP(ipToRelease *net.IP) error
}

type allocateClient struct {
	aws    *awsclient
	subnet SubnetsClient
}

// AllocateIPOn allocates an IP on a specific interface.
func (c *allocateClient) AllocateIPOn(intf Interface) (*AllocationResult, error) {
	client, err := c.aws.newEC2()
	if err != nil {
		return nil, err
	}
	request := ec2.AssignPrivateIpAddressesInput{
		NetworkInterfaceId: &intf.ID,
	}
	request.SetSecondaryPrivateIpAddressCount(1)

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
						return &AllocationResult{
							&newip,
							newIntf,
						}, nil
					}
				}
			}
		}
		time.Sleep(1.0 * time.Second)
	}

	return nil, fmt.Errorf("Can't locate new IP address from AWS")
}

// AllocateIPFirstAvailableAtIndex allocates an IP address, skipping any adapter < the given index
// Returns a reference to the interface the IP was allocated on
func (c *allocateClient) AllocateIPFirstAvailableAtIndex(index int) (*AllocationResult, error) {
	interfaces, err := c.aws.GetInterfaces()
	if err != nil {
		return nil, err
	}
	limits := c.aws.ENILimits()

	var candidates []Interface
	for _, intf := range interfaces {
		if intf.Number < index {
			continue
		}
		if len(intf.IPv4s) < limits.IPv4 {
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
				return c.AllocateIPOn(intf)
			}
		}
	}

	return nil, fmt.Errorf("Unable to allocate - no IPs available on any interfaces")
}

// AllocateIPFirstAvailable allocates an IP address on the first available IP address
// Returns a reference to the interface the IP was allocated on
func (c *allocateClient) AllocateIPFirstAvailable() (*AllocationResult, error) {
	return c.AllocateIPFirstAvailableAtIndex(0)
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
