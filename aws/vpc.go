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
	DescribeVPCPeerCIDRs(vpcID string) ([]*net.IPNet, error)
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

func (v *vpcCacheClient) DescribeVPCPeerCIDRs(vpcID string) (cidrs []*net.IPNet, err error) {
	key := fmt.Sprintf("vpc-peers-%v", vpcID)
	state := cache.Get(key, &cidrs)
	if state == cache.CacheFound {
		return
	}
	cidrs, err = v.vpc.DescribeVPCPeerCIDRs(vpcID)
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
			// Avoid adding non associated CIDRs to routes
			if *cblock.CidrBlockState.State != "associated" {
				continue
			}
			_, parsed, err := net.ParseCIDR(*cblock.CidrBlock)
			if err != nil {
				return nil, err
			}
			cidrs = append(cidrs, parsed)
		}
	}
	return cidrs, nil
}

// DescribeVPCPeerCIDRs returns a list of CIDRs for all peered VPCs to the given VPC
func (v *vpcclient) DescribeVPCPeerCIDRs(vpcID string) ([]*net.IPNet, error) {
	ec2c, err := v.aws.newEC2()
	if err != nil {
		return nil, err
	}

	req := &ec2.DescribeVpcPeeringConnectionsInput{}

	res, err := ec2c.DescribeVpcPeeringConnections(req)
	if err != nil {
		return nil, err
	}

	// In certain peering situations, a CIDR may be duplicated
	// and visible to the API, even if the CIDR is not active in
	// one of the peered VPCs. We store all of the CIDRs in a map
	// to de-duplicate them.
	cidrs := make(map[string]bool)

	for _, peering := range res.VpcPeeringConnections {
		var peer *ec2.VpcPeeringConnectionVpcInfo

		if vpcID == *peering.AccepterVpcInfo.VpcId {
			peer = peering.RequesterVpcInfo
		} else if vpcID == *peering.RequesterVpcInfo.VpcId {
			peer = peering.AccepterVpcInfo
		} else {
			continue
		}

		for _, cidrBlock := range peer.CidrBlockSet {
			_, _, err := net.ParseCIDR(*cidrBlock.CidrBlock)
			if err == nil {
				cidrs[*cidrBlock.CidrBlock] = true
			}
		}
	}

	var returnCidrs []*net.IPNet
	for cidrString := range cidrs {
		_, cidr, err := net.ParseCIDR(cidrString)
		if err == nil && cidr != nil {
			returnCidrs = append(returnCidrs, cidr)
		}
	}
	return returnCidrs, nil
}
