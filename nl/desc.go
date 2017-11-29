package nl

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
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

// With a heavy heart and deep internal sadness, we support Docker
func runningDockerContainers() (containerIds []string, err error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		containerIds = append(containerIds, container.ID)
	}
	return containerIds, nil
}

// GetIPs returns IPs allocated to interfaces, in all namespaces
// TODO: Remove addresses on control plane interfaces, filters
func GetIPs() ([]BoundIP, error) {

	originNs, err := netns.Get()
	if err != nil {
		return nil, err
	}
	defer originNs.Close()

	files, err := ioutil.ReadDir("/var/run/netns/")
	if err != nil {
		_, err = fmt.Fprintln(os.Stderr, "Couldn't enumerate named namespaces")
		if err != nil {
			// Ignore this error
		}
		files = []os.FileInfo{}
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
	for _, f := range files {
		nsHandle, err := netns.GetFromName(f.Name())
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

	// Check for running docker containers
	containers, err := runningDockerContainers()
	if err == nil {
		// Enter each docker container, get handles
		for _, f := range containers {
			nsHandle, err := netns.GetFromDocker(f)
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
	}

	return foundIps, nil
}
