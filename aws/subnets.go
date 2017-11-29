package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
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

// GetSubnetsForInstance returns a list of subnets for the running instance
func GetSubnetsForInstance() ([]Subnet, error) {
	var subnets []Subnet

	id, err := getIDDoc()
	if err != nil {
		return nil, err
	}
	az := id.AvailabilityZone

	ec2Client, err := newEC2()
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
