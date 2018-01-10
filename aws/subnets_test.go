package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type ec2SubnetsMock struct {
	ec2iface.EC2API
	Resp ec2.DescribeSubnetsOutput
}

func (e *ec2SubnetsMock) DescribeSubnets(in *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
	return &e.Resp, nil
}

func TestGetSubnetsForInstance(t *testing.T) {
	cases := []struct {
		Resp     ec2.DescribeSubnetsOutput
		Expected int
	}{
		{
			Expected: 1,
			Resp: ec2.DescribeSubnetsOutput{
				Subnets: []*ec2.Subnet{
					{
						AvailabilityZone:        aws.String("us-east-1a"),
						CidrBlock:               aws.String("192.168.0.0/24"),
						DefaultForAz:            aws.Bool(false),
						SubnetId:                aws.String("subnet-1234"),
						AvailableIpAddressCount: aws.Int64(12),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String("subnet 1"),
							},
						},
					},
				},
			},
		},
	}

	oldIDDoc := defaultClient.idDoc
	defer func() { defaultClient.idDoc = oldIDDoc }()

	defaultClient.idDoc = &ec2metadata.EC2InstanceIdentityDocument{
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1a",
	}

	for i, c := range cases {
		defaultClient.ec2Client = &ec2SubnetsMock{Resp: c.Resp}

		res, err := defaultClient.GetSubnetsForInstance()
		if err != nil {
			t.Fatalf("%d Mock returned an error - is it mocked? %v", i, err)
		}

		if len(res) != c.Expected {
			t.Fatalf("%d Subnets not all returned", i)
		}

	}
}
