package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lyft/cni-ipvlan-vpc-k8s/aws/cache"
	"github.com/pkg/errors"
)

// ENILimit contains limits for adapter count and addresses
type ENILimit struct {
	Adapters int64
	IPv4     int64
	IPv6     int64
}

// LimitsClient provides methods for locating limits in AWS
type LimitsClient interface {
	ENILimits() (*ENILimit, error)
}

var defaultLimit = ENILimit{
	Adapters: 4,
	IPv4:     15,
	IPv6:     15,
}

// ENILimitsForInstanceType returns the limits for ENI for an instance type
func (c *awsclient) ENILimitsForInstanceType(itype string) (*ENILimit, error) {
	client, err := c.newEC2()
	if err != nil {
		return nil, err
	}

	itypeList := []string{itype}
	describeInstanceTypesInput := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: aws.StringSlice(itypeList),
	}

	instanceDescribeOutput, err := client.DescribeInstanceTypes(describeInstanceTypesInput)
	if err != nil {
		return nil, err
	}
	if len(instanceDescribeOutput.InstanceTypes) == 0 {
		return nil, fmt.Errorf("empty answer from DescribeInstanceTypes for %s", itype)
	}

	netInfo := instanceDescribeOutput.InstanceTypes[0].NetworkInfo
	limit := &ENILimit{
		Adapters: *netInfo.MaximumNetworkInterfaces,
		IPv4:     *netInfo.Ipv4AddressesPerInterface,
		IPv6:     *netInfo.Ipv6AddressesPerInterface,
	}
	return limit, nil
}

// ENILimits returns the limits based on the system's instance type
func (c *awsclient) ENILimits() (*ENILimit, error) {
	id, err := c.getIDDoc()
	if err != nil || id == nil {
		return &defaultLimit, errors.Wrap(err, "unable get instance identity doc")
	}

	// Use the instance type in the cache key in case at some point the cache dir is persisted across reboots
	// (instances can be stopped and resized)
	key := "eni_limits_for_" + id.InstanceType
	limit := &ENILimit{}
	if cache.Get(key, limit) == cache.CacheFound {
		return limit, nil
	}

	limit, err = c.ENILimitsForInstanceType(id.InstanceType)
	if err != nil {
		return &defaultLimit, errors.Wrap(err, "unable get instance network limits")
	}

	cache.Store(key, 24*time.Hour, limit)
	return limit, nil
}
