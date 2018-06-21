package aws

import (
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/lyft/cni-ipvlan-vpc-k8s/nl"
)

var (
	interfacePollWaitTime         = 1000 * time.Millisecond
	interfaceSettleTime           = 30 * time.Second
	interfaceDetachWaitTime       = 1 * time.Second
	interfacePostDetachSettleTime = 5 * time.Second
	interfaceDetachAttempts       = 20 // interfaceDetachAttempts * interfaceDetachWaitTime = total wait time
)

// InterfaceClient provides methods for allocating and deallocating interfaces
type InterfaceClient interface {
	NewInterfaceOnSubnetAtIndex(index int, secGrps []string, subnet Subnet) (*Interface, error)
	NewInterface(secGrps []string, requiredTags map[string]string) (*Interface, error)
	RemoveInterface(interfaceIDs []string) error
}

type interfaceClient struct {
	aws    *awsclient
	subnet SubnetsClient
}

// NewInterfaceOnSubnetAtIndex creates a new Interface with a specified subnet and index
func (c *interfaceClient) NewInterfaceOnSubnetAtIndex(index int, secGrps []string, subnet Subnet) (*Interface, error) {
	client, err := c.aws.newEC2()
	if err != nil {
		return nil, err
	}

	idDoc, err := c.aws.getIDDoc()
	if err != nil {
		return nil, err
	}

	createReq := &ec2.CreateNetworkInterfaceInput{}
	createReq.SetDescription(fmt.Sprintf("CNI-ENI %v", idDoc.InstanceID))
	secGrpsPtr := []*string{}
	for _, grp := range secGrps {
		newgrp := grp // Need to copy
		secGrpsPtr = append(secGrpsPtr, &newgrp)
	}

	createReq.SetGroups(secGrpsPtr)
	createReq.SetSubnetId(subnet.ID)

	resp, err := client.CreateNetworkInterface(createReq)
	if err != nil {
		return nil, err
	}

	// resp.NetworkInterface.NetworkInterfaceId
	attachReq := &ec2.AttachNetworkInterfaceInput{}
	attachReq.SetDeviceIndex(int64(index))
	attachReq.SetInstanceId(idDoc.InstanceID)
	attachReq.SetNetworkInterfaceId(*resp.NetworkInterface.NetworkInterfaceId)

	attachResp, err := client.AttachNetworkInterface(attachReq)
	if err != nil {
		// We attempt to remove the interface we just made due to attachment failure
		delReq := &ec2.DeleteNetworkInterfaceInput{}
		delReq.SetNetworkInterfaceId(*resp.NetworkInterface.NetworkInterfaceId)

		_, delErr := client.DeleteNetworkInterface(delReq)
		if delErr != nil {
			return nil, delErr
		}
		return nil, err
	}

	// We have an attachment ID from the last API, which lets us mark the
	// interface as delete on termination
	changes := &ec2.NetworkInterfaceAttachmentChanges{}
	changes.SetAttachmentId(*attachResp.AttachmentId)
	changes.SetDeleteOnTermination(true)
	modifyReq := &ec2.ModifyNetworkInterfaceAttributeInput{}
	modifyReq.SetAttachment(changes)
	modifyReq.SetNetworkInterfaceId(*resp.NetworkInterface.NetworkInterfaceId)

	_, err = client.ModifyNetworkInterfaceAttribute(modifyReq)
	if err != nil {
		// Continue anyway
		fmt.Fprintf(os.Stderr,
			"Unable to mark interface for deletion due to %v",
			err)
	}

	for start := time.Now(); time.Since(start) <= interfaceSettleTime; time.Sleep(interfacePollWaitTime) {
		newInterfaces, err := c.aws.GetInterfaces()
		if err != nil {
			// The metadata server is inconsistent - for example, not
			// all of the nodes under the interface will populate at once
			// and instead return a 404 error. We just swallow this error here and
			// continue on.
			continue
		}
		for _, intf := range newInterfaces {
			if intf.Mac == *resp.NetworkInterface.MacAddress {
				// New IP. Timestamp the addition as a free IP.
				registry := &Registry{}
				registry.TrackIP(net.ParseIP(*resp.NetworkInterface.PrivateIpAddress))
				configureInterface(&intf)
				return &intf, nil
			}
		}

	}

	return nil, fmt.Errorf("interface did not attach in time")
}

// Fire and forget method to configure an interface
func configureInterface(intf *Interface) {
	// Found a match, going to try to make sure the interface is up
	err := nl.UpInterfacePoll(intf.LocalName())
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Interface %v could not be enabled. Networking will be broken.\n",
			intf.LocalName())
		return
	}
	baseMtu, err := nl.GetMtu("eth0")
	if err != nil || baseMtu < 1000 || baseMtu > 9001 {
		return
	}
	nl.SetMtu(intf.LocalName(), baseMtu)
}

// NewInterface creates an Interface based on specified parameters
func (c *interfaceClient) NewInterface(secGrps []string, requiredTags map[string]string) (*Interface, error) {
	subnets, err := c.subnet.GetSubnetsForInstance()
	if err != nil {
		return nil, err
	}

	existingInterfaces, err := c.aws.GetInterfaces()
	if err != nil {
		return nil, err
	}

	limits := c.aws.ENILimits()
	if len(existingInterfaces) >= limits.Adapters {
		return nil, fmt.Errorf("too many adapters on this instance already")
	}

	var availableSubnets []Subnet

OUTER:
	for _, newSubnet := range subnets {
		// Match incoming tags
		for tagKey, tagValue := range requiredTags {
			value, ok := newSubnet.Tags[tagKey]
			// Skip untagged subnets and ones not matching
			// the required tag
			if !ok || (ok && value != tagValue) {
				continue OUTER
			}
		}
		availableSubnets = append(availableSubnets, newSubnet)
	}

	// assign new interfaces to subnets with most available addresses
	sort.Sort(SubnetsByAvailableAddressCount(availableSubnets))

	if len(availableSubnets) <= 0 {
		return nil, fmt.Errorf("No subnets are available which haven't already been used")
	}

	return c.NewInterfaceOnSubnetAtIndex(len(existingInterfaces), secGrps, availableSubnets[0])
}

// RemoveInterface gracefull shutdown and removal of interfaces
// Simply detach the interface, wait for it to come down and then
// removes.
func (c *awsclient) RemoveInterface(interfaceIDs []string) error {
	client, err := c.newEC2()
	if err != nil {
		return err
	}

	for _, interfaceID := range interfaceIDs {
		// TODO: check if there is any other interface on this namespace?

		// We need the interface AttachmentId to detach
		interfaceDescription, err := c.describeNetworkInterface(interfaceID)
		if err != nil {
			return err
		}

		detachInterfaceInput := &ec2.DetachNetworkInterfaceInput{
			AttachmentId: interfaceDescription.Attachment.AttachmentId,
			DryRun:       aws.Bool(false),
			Force:        aws.Bool(false),
		}

		// Detach the networkinterface
		_, err = client.DetachNetworkInterface(detachInterfaceInput)
		if err != nil {
			fmt.Printf("Error occurced when trying to detach %v interface, use --force to override this check", interfaceID)
			return err
		}

		// Wait for the interface to be removed
		if err := c.waitUtilInterfaceDetaches(interfaceID); err != nil {
			return err
		}

		// Even after the interface detaches, you cannot delete right away
		time.Sleep(interfacePostDetachSettleTime)

		// Now we can safely remove the interface
		if err := c.deleteInterface(interfaceID); err != nil {
			return err
		}
	}
	return nil
}

func (c *awsclient) deleteInterface(interfaceID string) error {
	client, err := c.newEC2()
	if err != nil {
		return err
	}

	deleteInterfaceInput := &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(interfaceID),
	}

	_, err = client.DeleteNetworkInterface(deleteInterfaceInput)
	return err
}

func (c *awsclient) waitUtilInterfaceDetaches(interfaceID string) error {
	var interfaceDescription *ec2.NetworkInterface

	interfaceDescription, err := c.describeNetworkInterface(interfaceID)
	if err != nil {
		return err
	}

	// Once the ENI is in available state, we are ok to delete it
	for attempt := 0; *interfaceDescription.Status != "available"; attempt++ {
		interfaceDescription, err = c.describeNetworkInterface(interfaceID)
		if err != nil {
			return err
		}

		if attempt == interfaceDetachAttempts {
			return fmt.Errorf("Interface %v has not detached yet, use --force to override this check", interfaceID)
		}

		time.Sleep(interfaceDetachWaitTime)
	}

	return nil
}

func (c *awsclient) describeNetworkInterface(interfaceID string) (*ec2.NetworkInterface, error) {
	client, err := c.newEC2()
	if err != nil {
		return nil, err
	}

	interfaceIDList := []string{interfaceID}
	describeInterfaceInput := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: aws.StringSlice(interfaceIDList),
	}

	interfaceDescribeOutput, err := client.DescribeNetworkInterfaces(describeInterfaceInput)
	if err != nil {
		return nil, err
	}

	if len(interfaceDescribeOutput.NetworkInterfaces) <= 0 {
		return nil, fmt.Errorf("Cannot describe interface, it might not exist")
	}

	return interfaceDescribeOutput.NetworkInterfaces[0], nil
}
