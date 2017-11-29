package nl

import (
	"github.com/vishvananda/netlink"
)

// GetMtu gets the current MTU for an interface
func GetMtu(name string) (int, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return 0, err
	}
	return link.Attrs().MTU, nil
}

// SetMtu sets the MTU of an interface
func SetMtu(name string, mtu int) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetMTU(link, mtu)
}
