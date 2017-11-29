package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestNewEc2Filter(t *testing.T) {
	type FilterInput struct {
		Name   string
		Values []string
	}

	cases := []struct {
		Input  FilterInput
		Expect ec2.Filter
	}{
		{
			Input: FilterInput{
				Name:   "filter1",
				Values: []string{"value1"},
			},
			Expect: ec2.Filter{
				Name:   aws.String("filter1"),
				Values: aws.StringSlice([]string{"value1"}),
			},
		},
		{
			Input: FilterInput{
				Name:   "filter2",
				Values: []string{"value1", "value3", "value3"},
			},
			Expect: ec2.Filter{
				Name:   aws.String("filter2"),
				Values: aws.StringSlice([]string{"value1", "value3", "value3"}),
			},
		},
	}

	for i, c := range cases {
		filterOutput := newEc2Filter(c.Input.Name, c.Input.Values...)

		if reflect.DeepEqual(filterOutput, c.Expect) {
			t.Fatalf("%d newEc2Filter did not return expected results", i)
		}
	}
}
