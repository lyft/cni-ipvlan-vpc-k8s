package aws

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

// Interface describes an interface from the metadata service
type Interface struct {
	ID     string
	Mac    string
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
	return fmt.Sprintf("eth%d", i.Number)
}

// Interfaces contains a slice of Interface
type Interfaces []Interface

func (a Interfaces) Len() int           { return len(a) }
func (a Interfaces) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Interfaces) Less(i, j int) bool { return a[i].Number < a[j].Number }

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

func getInterface(mac string) (Interface, error) {
	var iface Interface
	iface.Mac = mac

	prefix := fmt.Sprintf("network/interfaces/macs/%s/", mac)
	get := func(val string) (data string, err error) {
		return metaData.GetMetadata(fmt.Sprintf("%s/%s", prefix, val))
	}
	metadataParser := func(metadataId string, modifer func(*Interface, string) error) error {
		metadata, _ := get(metadataId)
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
		for _, vpcCidr := range strings.Split(value, "\n") {
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

	return iface, nil
}

// Available returns the availability status
func Available() bool {
	return metaData.Available()
}

// GetInterfaces returns a list of configured interfaces
func GetInterfaces() ([]Interface, error) {
	var interfaces []Interface

	if !metaData.Available() {
		return nil, fmt.Errorf("EC2 Metadata not available")
	}

	macResult, err := metaData.GetMetadata("network/interfaces/macs/")
	if err != nil {
		return nil, err
	}

	macs := strings.Split(macResult, "\n")
	for _, mac := range macs {
		if len(mac) < 1 {
			continue
		}
		mac = mac[0 : len(mac)-1]
		iface, err := getInterface(mac)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, iface)
	}

	sort.Sort(Interfaces(interfaces))

	return interfaces, nil
}
