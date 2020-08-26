package nl

import (
	"os"
	"testing"

	"github.com/vishvananda/netlink"
)

func CreateTestInterface(t *testing.T, name string) {
	lyftBridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		TxQLen: -1,
		Name:   name,
	}}

	err := netlink.LinkAdd(lyftBridge)
	if err != nil {
		t.Errorf("Could not add %s: %v", lyftBridge.Name, err)
		err = RemoveInterface(lyftBridge.Name)
		if err != nil {
			t.Errorf("Failed to remove interface %s: %s", lyftBridge.Name, err)
		}
	}

	lyft1, _ := netlink.LinkByName(name)
	err = netlink.LinkSetMaster(lyft1, lyftBridge)
	if err != nil {
		t.Logf("Failed to set link master: %s", err)
	}
}

func TestUpInterface(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface(t, "lyft1")
	defer func() { _ = RemoveInterface("lyft1") }()

	if err := UpInterface("lyft1"); err != nil {
		t.Fatalf("Failed to UpInterface %v", err)
	}
}

func TestUpInterfacePoll(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface(t, "lyft2")
	defer func() { _ = RemoveInterface("lyft2") }()

	if err := UpInterfacePoll("lyft2"); err != nil {
		t.Fatalf("Failed to failed to stand up interface lyft2")
	}
}
