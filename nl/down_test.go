package nl

import (
	"os"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestDownInterface(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface(t, "lyft3")
	defer RemoveInterface("lyft3")

	if err := UpInterface("lyft3"); err != nil {
		t.Fatalf("Failed to UpInterface lyft3: %v", err)
	}

	if err := DownInterface("lyft3"); err != nil {
		t.Fatalf("Failed to DownInterface lyft3: %v", err)
	}
}

func TestRemoveInterface(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root or network capabilities - skipped")
		return
	}

	CreateTestInterface(t, "lyft4")

	if err := RemoveInterface("lyft4"); err != nil {
		t.Fatalf("Failed to RemoveInterface lyft4: %v", err)
	}

	// This link should not exist, this call should fail
	link, _ := netlink.LinkByName("lyft4")

	if link != nil {
		t.Fatal("Failed to RemoveInterface lyft4")
	}
}
