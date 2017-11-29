package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type ec2ClientMock struct {
	ec2iface.EC2API
	NetworkDescribeResponse ec2.DescribeNetworkInterfacesOutput
	NetworkDeleteResponse   ec2.DeleteNetworkInterfaceOutput
	NetworkDetachResponse   ec2.DetachNetworkInterfaceOutput
}

func (e *ec2ClientMock) DescribeNetworkInterfaces(in *ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &e.NetworkDescribeResponse, nil
}

func (e *ec2ClientMock) DeleteNetworkInterface(in *ec2.DeleteNetworkInterfaceInput) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &e.NetworkDeleteResponse, nil
}

func (e *ec2ClientMock) DetachNetworkInterface(in *ec2.DetachNetworkInterfaceInput) (*ec2.DetachNetworkInterfaceOutput, error) {
	return &e.NetworkDetachResponse, nil
}

// func TestNewInterfaceOnSubnetAtIndex(t *testing.T) {}
// func TestConfigureInterface(t *testing.T) {}
// func TestNewInterface(t *testing.T) {}

func TestRemoveInterface(t *testing.T) {
	interfaceDetachAttempts = 1
	interfacePostDetachSettleTime = 1

	cases := []struct {
		Input                   []string
		NetworkDescribeResponse ec2.DescribeNetworkInterfacesOutput
	}{
		{
			Input: []string{"eni-lyft-1"},
			NetworkDescribeResponse: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						Attachment: &ec2.NetworkInterfaceAttachment{
							AttachmentId: aws.String("eni-lyft-1-attachmentid"),
						},
						NetworkInterfaceId: aws.String("eni-lyft-1"),
						Status:             aws.String("available"),
					},
				},
			},
		},
		{
			Input: []string{"eni-lyft-1", "eni-lyft-2"},
			NetworkDescribeResponse: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						Attachment: &ec2.NetworkInterfaceAttachment{
							AttachmentId: aws.String("eni-lyft-1-attachmentid"),
						},
						NetworkInterfaceId: aws.String("eni-lyft-1"),
						Status:             aws.String("available"),
					},
					{
						Attachment: &ec2.NetworkInterfaceAttachment{
							AttachmentId: aws.String("eni-lyft-2-attachmentid"),
						},
						NetworkInterfaceId: aws.String("eni-lyft-2"),
						Status:             aws.String("pending"),
					},
				},
			},
		},
	}

	for i, c := range cases {
		_ec2Client = &ec2ClientMock{
			NetworkDescribeResponse: c.NetworkDescribeResponse,
		}
		err := RemoveInterface(c.Input)

		if err != nil {
			t.Fatalf("%d Mock returned an error: %v", i, err)
		}
	}
}

func TestDeleteInterface(t *testing.T) {
	cases := []struct {
		Input    string
		Response ec2.DeleteNetworkInterfaceOutput
	}{
		{
			Input:    "eni-lyft-1",
			Response: ec2.DeleteNetworkInterfaceOutput{},
		},
	}

	for i, c := range cases {
		_ec2Client = &ec2ClientMock{NetworkDeleteResponse: c.Response}
		err := deleteInterface(c.Input)

		if err != nil {
			t.Fatalf("%d Mock returned an error: %v", i, err)
		}
	}
}

func TestWaitUntilInterfaceDetaches(t *testing.T) {
	interfaceDetachAttempts = 1
	cases := []struct {
		Input    string
		Expected string
		Response ec2.DescribeNetworkInterfacesOutput
	}{
		{
			Input:    "eni-lyft-1",
			Expected: "",
			Response: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-lyft-1"),
						Status:             aws.String("available"),
					},
				},
			},
		},
		{
			Input:    "eni-lyft-2",
			Expected: "Interface eni-lyft-2 has not detached yet, use --force to override this check",
			Response: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-lyft-2"),
						Status:             aws.String("pending"),
					},
				},
			},
		},
	}

	for i, c := range cases {
		_ec2Client = &ec2ClientMock{NetworkDescribeResponse: c.Response}
		err := waitUtilInterfaceDetaches(c.Input)

		if err != nil {
			if err.Error() != c.Expected {
				t.Fatalf("%d Mock returned an error: %v", i, err)
			}
		}
	}
}

func TestDescribeNetworkInterface(t *testing.T) {
	cases := []struct {
		Input    string
		Expected ec2.NetworkInterface
		Response ec2.DescribeNetworkInterfacesOutput
	}{
		{
			Input: "eni-lyft-1",
			Expected: ec2.NetworkInterface{
				NetworkInterfaceId: aws.String("eni-lyft-1"),
			},
			Response: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-lyft-1"),
					},
				},
			},
		},
		{
			Input: "eni-lyft-2",
			// We actually dont expect anything here, this should throw an error as no
			// interfaces were returned
			Expected: ec2.NetworkInterface{
				NetworkInterfaceId: aws.String("eni-lyft-"),
			},
			Response: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{},
			},
		},
	}

	for i, c := range cases {
		_ec2Client = &ec2ClientMock{NetworkDescribeResponse: c.Response}
		res, err := describeNetworkInterface(c.Input)

		if err != nil {
			if err.Error() != "Cannot describe interface, it might not exist" {
				t.Fatalf("%d Mock returned an error: %v", i, err)
			}
		}

		if reflect.DeepEqual(res, c.Expected) {
			t.Fatalf("%d DescribeNetworkInterface did not return expected results", i)
		}
	}
}
