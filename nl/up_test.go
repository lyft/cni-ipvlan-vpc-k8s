package nl

import (
	"fmt"
	"testing"

	"github.com/vishvananda/netlink"
)

func CreateTestInterface(name string) {
	lyftBridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		TxQLen: -1,
		Name:   name,
	}}

	err := netlink.LinkAdd(lyftBridge)
	if err != nil {
		RemoveInterface(name)
		fmt.Errorf("Could not add %s: %v", lyftBridge.Name, err)
	}

	lyft1, _ := netlink.LinkByName(name)
	netlink.LinkSetMaster(lyft1, lyftBridge)
}

func TestUpInterface(t *testing.T) {
	CreateTestInterface("lyft1")
	defer RemoveInterface("lyft1")

	if err := UpInterface("lyft1"); err != nil {
		t.Fatalf("Failed to UpInterface %v", err)
	}
}

func TestUpInterfacePoll(t *testing.T) {
	CreateTestInterface("lyft2")
	defer RemoveInterface("lyft2")

	if err := UpInterfacePoll("lyft2"); err != nil {
		t.Fatalf("Failed to failed to stand up interface lyft2")
	}
}
