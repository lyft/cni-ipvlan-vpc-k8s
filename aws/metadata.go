package aws

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
)

// Interface describes an interface from the metadata service
type Interface struct {
	ID     string
	Mac    string
	IfName string
	Number int
	IPv4s  []net.IP

	SubnetID   string
	SubnetCidr *net.IPNet

	VpcID            string
	VpcPrimaryCidr   *net.IPNet
	VpcCidrs         []*net.IPNet
	SecurityGroupIds []string
}

// LocalName returns the instance name in string form
func (i Interface) LocalName() string {
	return i.IfName
}

// Interfaces contains a slice of Interface
type Interfaces []Interface

func (a Interfaces) Len() int           { return len(a) }
func (a Interfaces) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Interfaces) Less(i, j int) bool { return a[i].Number < a[j].Number }

// MetadataClient provides methods to query the metadata service
type MetadataClient interface {
	Available() bool
	GetInterfaces() ([]Interface, error)
	InstanceType() string
}

// EC2 generally gives the following data blocks from an interface in meta-data
// device-number
// interface-id
// local-hostname
// local-ipv4s
// mac
// owner-id
// security-group-ids
// security-groups
// subnet-id
// subnet-ipv4-cidr-block
// vpc-id
// vpc-ipv4-cidr-block
// vpc-ipv4-cidr-blocks
// vpc-ipv6-cidr-blocks

func (c *awsclient) getInterface(mac string) (Interface, error) {
	var iface Interface
	iface.Mac = mac

	prefix := fmt.Sprintf("network/interfaces/macs/%s/", mac)
	get := func(val string) (data string, err error) {
		return c.metaData.GetMetadata(fmt.Sprintf("%s/%s", prefix, val))
	}
	metadataParser := func(metadataId string, modifer func(*Interface, string) error) error {
		metadata, err := get(metadataId)
		if err != nil {
			log.Printf("Error calling metadata service: %v", err)
			return err
		}
		if metadata != "" {
			return modifer(&iface, metadata)
		}
		return nil
	}

	if err := metadataParser("interface-id", func(iface *Interface, value string) error {
		iface.ID = value
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("device-number", func(iface *Interface, value string) error {
		num, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		iface.Number = num
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("local-ipv4s", func(iface *Interface, value string) error {
		for _, ipv4 := range strings.Split(value, "\n") {
			parsed := net.ParseIP(ipv4)
			if parsed != nil {
				iface.IPv4s = append(iface.IPv4s, parsed)
			}
		}
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("subnet-id", func(iface *Interface, value string) error {
		iface.SubnetID = value
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("subnet-ipv4-cidr-block", func(iface *Interface, value string) error {
		var err error
		_, iface.SubnetCidr, err = net.ParseCIDR(value)
		return err
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("vpc-id", func(iface *Interface, value string) error {
		iface.VpcID = value
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("vpc-ipv4-cidr-block", func(iface *Interface, value string) error {
		var err error
		_, iface.VpcPrimaryCidr, err = net.ParseCIDR(value)
		return err
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("vpc-ipv4-cidr-blocks", func(iface *Interface, value string) error {
		cidrList := strings.Split(value, "\n")
		if len(cidrList) == 0 {
			return fmt.Errorf("No VPC ranges found")
		}
		for _, vpcCidr := range cidrList {
			_, net, err := net.ParseCIDR(vpcCidr)
			if err != nil {
				return err
			}
			iface.VpcCidrs = append(iface.VpcCidrs, net)
		}
		return nil
	}); err != nil {
		return iface, err
	}

	if err := metadataParser("security-group-ids", func(iface *Interface, value string) error {
		secGrps := strings.Split(value, "\n")
		iface.SecurityGroupIds = secGrps
		return nil
	}); err != nil {
		return iface, err
	}

	// Retrieve interface name on host for this MAC address
	ifaces, err := net.Interfaces()
	if err != nil {
		return iface, err
	}
	for _, i := range ifaces {
		if i.HardwareAddr.String() == mac {
			iface.IfName = i.Name
			break
		}
	}
	// Commented because the AWS metadata server can return MAC addreses from detached interfaces on c5/m5
	// A cleaner fix would be to ignore bogus interfaces (but probably not the effort because it should get fixed soon)
	//if iface.IfName  == "" {
	//	return iface, fmt.Errorf("Unable to locate interface with mac %s on host", mac)
	//}

	return iface, nil
}

// Available returns the availability status
func (c *awsclient) Available() bool {
	return c.metaData.Available()
}

// GetInterfaces returns a list of configured interfaces
func (c *awsclient) GetInterfaces() ([]Interface, error) {
	var interfaces []Interface

	if !c.metaData.Available() {
		return nil, fmt.Errorf("EC2 Metadata not available")
	}

	macResult, err := c.metaData.GetMetadata("network/interfaces/macs/")
	if err != nil {
		return nil, err
	}

	macs := strings.Split(macResult, "\n")
	for _, mac := range macs {
		if len(mac) < 1 {
			continue
		}
		mac = mac[0 : len(mac)-1]
		iface, err := c.getInterface(mac)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, iface)
	}

	sort.Sort(Interfaces(interfaces))

	return interfaces, nil
}

// InstanceType gets the type of the instance, i.e. "c5.large"
func (c *awsclient) InstanceType() string {
	id, err := c.getIDDoc()
	if err != nil {
		return "unknown"
	}

	return id.InstanceType
}
