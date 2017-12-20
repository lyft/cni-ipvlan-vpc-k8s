package main

import (
	"testing"
)

// TestFilterBuildNil checks the empty string input
func TestFilterBuildNil(t *testing.T) {
	ret, err := filterBuild("")
	if ret != nil && err != nil {
		t.Errorf("Nil input wasn't nil")
	}
}

// TestFilterBuildSingle tests a single filter
func TestFilterBuildSingle(t *testing.T) {
	ret, err := filterBuild("foo=bar")
	if err != nil {
		t.Errorf("Error returned")
	}
	if v, ok := ret["foo"]; !ok || v != "bar" {
		t.Errorf("Invalid return from filter")
	}
}

// TestFilterBuildMulti tests multiple filters
func TestFilterBuildMulti(t *testing.T) {
	ret, err := filterBuild("foo=bar,err=frr")
	if err != nil {
		t.Errorf("Error returned")
	}
	if v, ok := ret["foo"]; !ok || v != "bar" {
		t.Errorf("Invalid return from filter - no foo")
	}
	if v, ok := ret["err"]; !ok || v != "frr" {
		t.Errorf("Invalid return from filter - no frr")
	}

}
