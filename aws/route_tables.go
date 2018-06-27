package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lyft/cni-ipvlan-vpc-k8s/aws/cache"
	"github.com/lyft/cni-ipvlan-vpc-k8s/lib"
	"net"
	"strings"
	"time"
)

// RouteTablesClient reads routes from the VPC route table associated with a subnet
type RouteTablesClient interface {
	GetRoutesForSubnet(string) ([]*net.IPNet, error)
}

type routeTablesCacheClient struct {
	routeTables *routeTablesClient
	expiration  time.Duration
}

func (r *routeTablesCacheClient) GetRoutesForSubnet(subnetID string) ([]*net.IPNet, error) {
	key := fmt.Sprintf("routes_for_subnet_%s", subnetID)
	var routes []*net.IPNet
	state := cache.Get(key, routes)
	if state == cache.CacheFound {
		return routes, nil
	}
	routes, err := r.routeTables.GetRoutesForSubnet(subnetID)
	if err != nil {
		return nil, err
	}
	cache.Store(key, r.expiration, &routes)
	return routes, nil
}

type routeTablesClient struct {
	aws *awsclient
}

func (r *routeTablesClient) GetRoutesForSubnet(subnetID string) ([]*net.IPNet, error) {
	ec2Client, err := r.aws.newEC2()
	if err != nil {
		return nil, err
	}

	req := &ec2.DescribeRouteTablesInput{}
	req.Filters = []*ec2.Filter{
		newEc2Filter("association.subnet-id", subnetID),
	}
	res, err := ec2Client.DescribeRouteTables(req)
	if err != nil {
		return nil, err
	}

	var cidrs []*net.IPNet
	for _, table := range res.RouteTables {
		for _, route := range table.Routes {
			cidr, err := getCidr(route)
			if cidr == nil || err != nil || isDefaultRoute(cidr) || targetIsInternetGateway(route) {
				continue
			}
			cidrs = append(cidrs, cidr)
		}
	}
	return lib.DeduplicateCidrs(cidrs), nil
}

// Check if the target is an internet gateway (which would require a public ip
// to be bound to the interface in order to use)
func targetIsInternetGateway(r *ec2.Route) bool {
	if r.EgressOnlyInternetGatewayId != nil {
		return true
	}
	if r.GatewayId != nil && strings.HasPrefix(*r.GatewayId, "igw-") {
		return true
	}
	return false
}

// Check if a cidr is the IPv4 or IPv6 default route
func isDefaultRoute(cidr *net.IPNet) bool {
	return cidr.IP.IsUnspecified()
}

// Read and parse the IPv4 or IPv6 cidr from an EC2 route (whichever is present)
func getCidr(r *ec2.Route) (*net.IPNet, error) {
	var cidrString string
	if r.DestinationCidrBlock != nil {
		cidrString = *r.DestinationCidrBlock
	} else if r.DestinationIpv6CidrBlock != nil {
		cidrString = *r.DestinationIpv6CidrBlock
	} else {
		return nil, nil
	}
	_, cidr, err := net.ParseCIDR(cidrString)
	return cidr, err
}
