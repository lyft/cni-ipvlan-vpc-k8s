package nl

import (
	"fmt"
	"os"
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
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface("lyft1")
	defer RemoveInterface("lyft1")

	if err := UpInterface("lyft1"); err != nil {
		t.Fatalf("Failed to UpInterface %v", err)
	}
}

func TestUpInterfacePoll(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface("lyft2")
	defer RemoveInterface("lyft2")

	if err := UpInterfacePoll("lyft2"); err != nil {
		t.Fatalf("Failed to failed to stand up interface lyft2")
	}
}
