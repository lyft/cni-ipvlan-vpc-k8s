package aws

import (
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/lyft/cni-ipvlan-vpc-k8s/aws/cache"
)

// VPCClient provides a view into a VPC
type VPCClient interface {
	DescribeVPCCIDRs(vpcID string) ([]*net.IPNet, error)
}

type vpcCacheClient struct {
	vpc        *vpcclient
	expiration time.Duration
}

func (v *vpcCacheClient) DescribeVPCCIDRs(vpcID string) (cidrs []*net.IPNet, err error) {
	key := fmt.Sprintf("vpc-cidr-%v", vpcID)
	state := cache.Get(key, &cidrs)
	if state == cache.CacheFound {
		return
	}
	cidrs, err = v.vpc.DescribeVPCCIDRs(vpcID)
	if err != nil {
		return nil, err
	}
	cache.Store(key, v.expiration, &cidrs)
	return
}

type vpcclient struct {
	aws *awsclient
}

// DescribeVPCCIDRs returns a list of all CIDRS associated with a VPC
func (v *vpcclient) DescribeVPCCIDRs(vpcID string) ([]*net.IPNet, error) {
	req := &ec2.DescribeVpcsInput{
		VpcIds: []*string{aws.String(vpcID)},
	}
	ec2, err := v.aws.newEC2()
	if err != nil {
		return nil, err
	}
	res, err := ec2.DescribeVpcs(req)
	if err != nil {
		return nil, err
	}

	var cidrs []*net.IPNet

	for _, vpc := range res.Vpcs {
		for _, cblock := range vpc.CidrBlockAssociationSet {
			_, parsed, err := net.ParseCIDR(*cblock.CidrBlock)
			if err != nil {
				return nil, err
			}
			cidrs = append(cidrs, parsed)
		}
	}
	return cidrs, nil
}
