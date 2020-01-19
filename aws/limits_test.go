package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type ec2InstanceTypesMock struct {
	ec2iface.EC2API
	InstanceTypesDescribeResp  ec2.DescribeInstanceTypesOutput
	InstanceTypesDescribeCalls int
}

func (e *ec2InstanceTypesMock) DescribeInstanceTypes(in *ec2.DescribeInstanceTypesInput) (*ec2.DescribeInstanceTypesOutput, error) {
	e.InstanceTypesDescribeCalls++
	return &e.InstanceTypesDescribeResp, nil
}

func TestLimitsReturn(t *testing.T) {
	oldIDDoc := defaultClient.idDoc
	defer func() { defaultClient.idDoc = oldIDDoc }()

	cases := []struct {
		iType    string
		Resp     ec2.DescribeInstanceTypesOutput
		Expected *ENILimit
	}{
		{
			Expected: &ENILimit{
				Adapters: 4,
				IPv4:     15,
				IPv6:     15,
			},
			iType: "r4.xlarge",
			Resp: ec2.DescribeInstanceTypesOutput{
				InstanceTypes: []*ec2.InstanceTypeInfo{
					{
						NetworkInfo: &ec2.NetworkInfo{
							Ipv4AddressesPerInterface: aws.Int64(15),
							Ipv6AddressesPerInterface: aws.Int64(15),
							MaximumNetworkInterfaces:  aws.Int64(4),
						},
					},
				},
			},
		},
		{
			Expected: &ENILimit{
				Adapters: 15,
				IPv4:     50,
				IPv6:     50,
			},
			iType: "c5n.18xlarge",
			Resp: ec2.DescribeInstanceTypesOutput{
				InstanceTypes: []*ec2.InstanceTypeInfo{
					{
						NetworkInfo: &ec2.NetworkInfo{
							Ipv4AddressesPerInterface: aws.Int64(50),
							Ipv6AddressesPerInterface: aws.Int64(50),
							MaximumNetworkInterfaces:  aws.Int64(15),
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		defaultClient.idDoc = &ec2metadata.EC2InstanceIdentityDocument{
			Region:           "us-east-1",
			AvailabilityZone: "us-east-1a",
			InstanceType:     c.iType,
		}
		mock := &ec2InstanceTypesMock{
			InstanceTypesDescribeResp: c.Resp,
		}
		defaultClient.ec2Client = mock

		res, _ := defaultClient.ENILimits()
		if !reflect.DeepEqual(res, c.Expected) {
			t.Fatalf("ENILimits do not match. Expected: %v Got: %v", c.Expected, res)
		}

		calls := mock.InstanceTypesDescribeCalls

		res, _ = defaultClient.ENILimits()
		if mock.InstanceTypesDescribeCalls != calls {
			t.Fatalf("Caching logic failed, API call made")
		}
		if !reflect.DeepEqual(res, c.Expected) {
			t.Fatalf("ENILimits from cache do not match. Expected: %v Got: %v", c.Expected, res)
		}
	}
}
