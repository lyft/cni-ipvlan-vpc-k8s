package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lyft/cni-ipvlan-vpc-k8s/aws/cache"
	"time"
)

// Subnet contains attributes of a subnet
type Subnet struct {
	ID                    string
	Cidr                  string
	IsDefault             bool
	AvailableAddressCount int
	Name                  string
	Tags                  map[string]string
}

// SubnetsByAvailableAddressCount contains a list of subnet
type SubnetsByAvailableAddressCount []Subnet

func (a SubnetsByAvailableAddressCount) Len() int      { return len(a) }
func (a SubnetsByAvailableAddressCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SubnetsByAvailableAddressCount) Less(i, j int) bool {
	return a[i].AvailableAddressCount > a[j].AvailableAddressCount
}

// SubnetsClient provides information about VPC subnets
type SubnetsClient interface {
	GetSubnetsForInstance() ([]Subnet, error)
}

type subnetsCacheClient struct {
	subnets    *subnetsClient
	expiration time.Duration
}

func (s *subnetsCacheClient) GetSubnetsForInstance() (subnets []Subnet, err error) {
	state := cache.Get("subnets_for_instance", &subnets)
	if state == cache.CacheFound {
		return
	}
	subnets, err = s.subnets.GetSubnetsForInstance()
	cache.Store("subnets_for_instance", s.expiration, &subnets)
	return
}

type subnetsClient struct {
	aws *awsclient
}

// GetSubnetsForInstance returns a list of subnets for the running instance
func (c *subnetsClient) GetSubnetsForInstance() ([]Subnet, error) {
	var subnets []Subnet

	id, err := c.aws.getIDDoc()
	if err != nil {
		return nil, err
	}
	az := id.AvailabilityZone

	ec2Client, err := c.aws.newEC2()
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeSubnetsInput{}
	input.Filters = []*ec2.Filter{newEc2Filter("availabilityZone", az)}
	result, err := ec2Client.DescribeSubnets(input)

	if err != nil {
		return nil, err
	}

	for _, awsSub := range result.Subnets {
		subnet := Subnet{
			ID:                    *awsSub.SubnetId,
			Cidr:                  *awsSub.CidrBlock,
			IsDefault:             *awsSub.DefaultForAz,
			AvailableAddressCount: int(*awsSub.AvailableIpAddressCount),
			Tags: map[string]string{},
		}
		// Set all the tags on the result
		for _, tag := range awsSub.Tags {
			if *tag.Key == "Name" {
				subnet.Name = *tag.Value
			} else {
				subnet.Tags[*tag.Key] = *tag.Value
			}
		}
		subnets = append(subnets, subnet)
	}

	return subnets, nil
}
