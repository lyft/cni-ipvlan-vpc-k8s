package nl

import (
	"fmt"
	"os"
	"time"

	"github.com/vishvananda/netlink"
)

const interfaceSettleWaitTime = 100 * time.Millisecond
const interfaceSettleDeadline = 20 * time.Second

// UpInterface brings up an interface by name
func UpInterface(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(link)
}

// UpInterfacePoll waits until an interface can be resolved by netlink and then call up on the interface.
func UpInterfacePoll(name string) error {
	for start := time.Now(); time.Since(start) <= interfaceSettleDeadline; time.Sleep(interfaceSettleWaitTime) {
		err := UpInterface(name)
		if err == nil {
			return nil
		}
		_, err = fmt.Fprintf(os.Stderr, "Failing to enumerate %v due to %v\n", name, err)
		if err != nil {
			panic(err)
		}
	}
	return fmt.Errorf("Interface was not found after setting time")
}
