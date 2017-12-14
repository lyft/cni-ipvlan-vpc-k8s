package nl

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// BoundIP contains an IPNet / Label pair
type BoundIP struct {
	*net.IPNet
	Label string
}

func getIpsOnHandle(handle *netlink.Handle) ([]BoundIP, error) {
	foundIps := []BoundIP{}

	links, err := handle.LinkList()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		addrs, err := handle.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := *addr.IPNet
			found := BoundIP{
				&ip,
				addr.Label,
			}
			foundIps = append(foundIps, found)
		}

	}
	return foundIps, nil
}

// GetIPs returns IPs allocated to interfaces, in all namespaces
// TODO: Remove addresses on control plane interfaces, filters
func GetIPs() ([]BoundIP, error) {

	originNs, err := netns.Get()
	if err != nil {
		return nil, err
	}
	defer originNs.Close()

	var namespaces []string

	files, err := ioutil.ReadDir("/var/run/netns/")
	if err != nil {
		_, err = fmt.Fprintln(os.Stderr, "Couldn't enumerate named namespaces")
		if err != nil {
			// Ignore this error
		}
		files = []os.FileInfo{}
	} else {
		for _, file := range files {
			namespaces = append(namespaces,
				fmt.Sprintf("/var/run/netns/%s", file.Name()))
		}
	}

	// Check for running docker containers
	containers, err := runningDockerContainers()
	if err == nil {
		dockerNamespaces := dockerNetworkNamespaces(containers)
		namespaces = append(namespaces, dockerNamespaces...)
	}

	// First get all the IPs on the first handle
	handle, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}
	foundIps, err := getIpsOnHandle(handle)
	if err != nil {
		return nil, err
	}

	// Enter each _named_ namesapce, get handles
	for _, f := range namespaces {
		nsHandle, err := netns.GetFromPath(f)
		if err != nil {
			return nil, err
		}
		handle, err = netlink.NewHandleAt(nsHandle)
		if err != nil {
			return nil, err
		}

		newIps, err := getIpsOnHandle(handle)
		if err != nil {
			return nil, err
		}
		foundIps = append(foundIps, newIps...)

		handle.Delete()
		err = nsHandle.Close()
		if err != nil {
			panic(err)
		}
	}

	return foundIps, nil
}
