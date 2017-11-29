package nl

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// DownInterface Takes down a single interface
func DownInterface(name string) error {
	link, err := netlink.LinkByName(name)

	if err != nil {
		return err
	}

	if err := netlink.LinkSetDown(link); err != nil {
		fmt.Printf("Unable to set interface %v down", name)
		return err
	}

	return nil
}

// RemoveInterface complete removes the interface
func RemoveInterface(name string) error {
	link, err := netlink.LinkByName(name)

	if err != nil {
		return err
	}

	if err := netlink.LinkDel(link); err != nil {
		fmt.Printf("Unable to remove interface %v", name)
		return err
	}

	return nil
}
