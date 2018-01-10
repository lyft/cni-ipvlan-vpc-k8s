package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

func TestLimitsReturn(t *testing.T) {
	oldIDDoc := defaultClient.idDoc
	defer func() { defaultClient.idDoc = oldIDDoc }()

	defaultClient.idDoc = &ec2metadata.EC2InstanceIdentityDocument{
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1a",
		InstanceType:     "r4.xlarge",
	}

	limits := defaultClient.ENILimits()
	if limits.Adapters != 4 && limits.IPv4 != 15 {
		t.Fatalf("No valid limit returned for r4.xlarge %v", limits)
	}
}
