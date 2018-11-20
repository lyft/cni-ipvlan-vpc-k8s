package aws

import (
	"strings"
)

// Bug defines a bug name and a function to check if the
// system is affected
type Bug struct {
	Name   string
	HasBug func() bool
}

// ListBugs returns an enumerated set of all known bugs in AWS or instances
// that this instance is afflicted by
func ListBugs(meta MetadataClient) []Bug {
	bugs := []Bug{
		{
			Name:   "broken_cidr",
			HasBug: func() bool { return HasBugBrokenVPCCidrs(meta) },
		},
	}
	return bugs
}

// HasBugBrokenVPCCidrs returns true if this instance has a known defective
// meta-data service which will not return secondary VPC CIDRs. As of January 2018,
// this covers c5 and m5 class hardware.
func HasBugBrokenVPCCidrs(meta MetadataClient) bool {
	itype := meta.InstanceType()
	family := strings.Split(itype, ".")[0]
	switch family {
	case "c5", "m5", "c5d", "m5d", "m5a", "r5", "r5d", "r5a":
		return true
	default:
		return false
	}
}
