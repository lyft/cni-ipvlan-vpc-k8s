package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

func TestClientCreate(t *testing.T) {
	oldIDDoc := defaultClient.idDoc
	defer func() { defaultClient.idDoc = oldIDDoc }()

	defaultClient.idDoc = &ec2metadata.EC2InstanceIdentityDocument{
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1a",
	}

	client, err := defaultClient.newEC2()
	if err != nil {
		t.Errorf("Error generated %v", err)
	}

	if client == nil {
		t.Errorf("No client returned %v", err)
	}

	client2, err := defaultClient.newEC2()
	if client != client2 || err != nil {
		t.Errorf("Clients returned were not identical (no caching)")
	}

}
