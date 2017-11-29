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

// AllocateIPOn allocates an IP on a specific interface.
func AllocateIPOn(intf Interface) (*AllocationResult, error) {
	client, err := newEC2()
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

	for attempts := 10; attempts > 0; attempts-- {
		newIntf, err := getInterface(intf.Mac)
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
					// New IP
					return &AllocationResult{
						&newip,
						newIntf,
					}, nil
				}
			}
		}
		time.Sleep(1.0 * time.Second)
	}

	return nil, fmt.Errorf("Can't locate new IP address from AWS")
}

// AllocateIPFirstAvailableAtIndex allocates an IP address, skipping any adapter < the given index
// Returns a reference to the interface the IP was allocated on
func AllocateIPFirstAvailableAtIndex(index int) (*AllocationResult, error) {
	interfaces, err := GetInterfaces()
	if err != nil {
		return nil, err
	}
	limits := ENILimits()

	var candidates []Interface
	for _, intf := range interfaces {
		if intf.Number < index {
			continue
		}
		if len(intf.IPv4s) < limits.IPv4 {
			candidates = append(candidates, intf)
		}
	}

	subnets, err := GetSubnetsForInstance()
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
				return AllocateIPOn(intf)
			}
		}
	}

	return nil, fmt.Errorf("Unable to allocate - no IPs available on any interfaces")
}

// AllocateIPFirstAvailable allocates an IP address on the first available IP address
// Returns a reference to the interface the IP was allocated on
func AllocateIPFirstAvailable() (*AllocationResult, error) {
	return AllocateIPFirstAvailableAtIndex(0)
}

// DeallocateIP releases an IP back to AWS
func DeallocateIP(ipToRelease *net.IP) error {
	client, err := newEC2()
	if err != nil {
		return err
	}
	interfaces, err := GetInterfaces()
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
