package lib

import (
	"reflect"
	"testing"
)

func TestDeduplicateCidrs(t *testing.T) {
	cidrs, err := ParseCidrs("192.168.1.1/24", "192.168.2.1/24", "192.168.1.1/32", "192.168.2.1/24")
	if err != nil {
		t.Error(err)
	}
	deduplicated := DeduplicateCidrs(cidrs)
	expected, err := ParseCidrs("192.168.1.1/24", "192.168.2.1/24")
	if err != nil {
		t.Error(err)
	}
	if reflect.DeepEqual(deduplicated, expected) {
		t.Errorf("%v != %v", deduplicated, expected)
	}
}
