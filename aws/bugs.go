package aws

import (
	"strings"
)

// HasBugBrokenVPCCidrs returns true if this instance has a known defective
// meta-data service which will not return secondary VPC CIDRs. As of January 2018,
// this covers c5 and m5 class hardware.
func HasBugBrokenVPCCidrs(meta MetadataClient) bool {
	itype := meta.InstanceType()
	family := strings.Split(itype, ".")[0]
	switch family {
	case "c5", "m5":
		return true
	default:
		return false
	}
}
