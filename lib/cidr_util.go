package lib

import (
	"net"
)

// DeduplicateCidrs deduplicates the entries in a list of net.IPNet and returns
// a new list. Note that this just deduplicates cidrs, it does not compact them.
func DeduplicateCidrs(inputCidrs []*net.IPNet) []*net.IPNet {
	cidrs := make(map[string]bool)
	for _, cidr := range inputCidrs {
		cidrs[cidr.String()] = true
	}
	var returnCidrs []*net.IPNet
	for cidrString := range cidrs {
		_, cidr, err := net.ParseCIDR(cidrString)
		if err == nil && cidr != nil {
			returnCidrs = append(returnCidrs, cidr)
		}
	}
	return returnCidrs
}

// ParseCidrs parses a list of cidr strings into a list of net.IPNet
func ParseCidrs(cidrStrings ...string) ([]*net.IPNet, error) {
	var returnCidrs []*net.IPNet
	for _, cidrString := range cidrStrings {
		_, cidr, err := net.ParseCIDR(cidrString)
		if err != nil {
			return nil, err
		}
		returnCidrs = append(returnCidrs, cidr)
	}
	return returnCidrs, nil
}
