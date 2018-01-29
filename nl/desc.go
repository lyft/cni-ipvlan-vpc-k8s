package nl

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
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

	var namespaces []string

	files, err := ioutil.ReadDir("/var/run/netns/")
	if err == nil {
		for _, file := range files {
			namespaces = append(namespaces,
				filepath.Join("/var/run/netns", file.Name()))
		}
	}

	// Check for running docker containers
	containers, err := runningDockerContainers()
	if err == nil {
		dockerNamespaces := dockerNetworkNamespaces(containers)
		namespaces = append(namespaces, dockerNamespaces...)
	}

	// First get all the IPs in the main namespace
	handle, err := netlink.NewHandle()
	if err != nil {
		return nil, err
	}
	foundIps, err := getIpsOnHandle(handle)
	if err != nil {
		return nil, err
	}

	// Enter each namesapce, get handles
	for _, nsPath := range namespaces {
		err := ns.WithNetNSPath(nsPath, func(_ ns.NetNS) error {

			handle, err = netlink.NewHandle()
			if err != nil {
				return err
			}
			defer handle.Delete()

			newIps, err := getIpsOnHandle(handle)
			foundIps = append(foundIps, newIps...)
			return err
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Enumerating namespace failure %v", err)
		}
	}

	return foundIps, nil
}
