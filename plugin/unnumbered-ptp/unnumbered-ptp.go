// Copyright 2017 CNI authors
// Copyright 2017 Lyft Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This is a sample chained plugin that supports multiple CNI versions. It
// parses prevResult according to the cniVersion
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils"
	"github.com/j-keck/arping"
	"github.com/vishvananda/netlink"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

// PluginConf is whatever you expect your configuration json to be. This is whatever
// is passed in on stdin. Your plugin may wish to expose its functionality via
// runtime args, see CONVENTIONS.md in the CNI spec.
type PluginConf struct {
	types.NetConf

	// This is the previous result, when called in the context of a chained
	// plugin. Because this plugin supports multiple versions, we'll have to
	// parse this in two passes. If your plugin is not chained, this can be
	// removed (though you may wish to error if a non-chainable plugin is
	// chained.
	// If you need to modify the result before returning it, you will need
	// to actually convert it to a concrete versioned struct.
	RawPrevResult *map[string]interface{} `json:"prevResult"`
	PrevResult    *current.Result         `json:"-"`

	IPMasq             bool   `json:"ipMasq"`
	HostInterface      string `json:"hostInterface"`
	ContainerInterface string `json:"containerInterface"`
	MTU                int    `json:"mtu"`
}

// parseConfig parses the supplied configuration (and prevResult) from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	// Parse previous result.
	if conf.RawPrevResult != nil {
		resultBytes, err := json.Marshal(conf.RawPrevResult)
		if err != nil {
			return nil, fmt.Errorf("could not serialize prevResult: %v", err)
		}
		res, err := version.NewResult(conf.CNIVersion, resultBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse prevResult: %v", err)
		}
		conf.RawPrevResult = nil
		conf.PrevResult, err = current.NewResultFromResult(res)
		if err != nil {
			return nil, fmt.Errorf("could not convert result to current version: %v", err)
		}
	}
	// End previous result parsing

	if conf.HostInterface == "" {
		return nil, fmt.Errorf("hostInterface must be specified")
	}

	if conf.ContainerInterface == "" {
		return nil, fmt.Errorf("containerInterface must be specified")
	}

	return &conf, nil
}

func setupContainerVeth(netns ns.NetNS, ifName string, mtu int, hostAddrs []netlink.Addr, pr *current.Result) (*current.Interface, *current.Interface, error) {
	hostInterface := &current.Interface{}
	containerInterface := &current.Interface{}

	err := netns.Do(func(hostNS ns.NetNS) error {
		hostVeth, contVeth0, err := ip.SetupVeth(ifName, mtu, hostNS)
		if err != nil {
			return err
		}
		hostInterface.Name = hostVeth.Name
		hostInterface.Mac = hostVeth.HardwareAddr.String()
		containerInterface.Name = contVeth0.Name
		containerInterface.Mac = contVeth0.HardwareAddr.String()
		containerInterface.Sandbox = netns.Path()

		for _, ipc := range pr.IPs {
			// All addresses apply to the container veth interface
			ipc.Interface = current.Int(1)
		}

		pr.Interfaces = []*current.Interface{hostInterface, containerInterface}

		contVeth, err := net.InterfaceByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to look up %q: %v", ifName, err)
		}

		// add host routes for each dst hostInterface ip on dev contVeth
		for _, ipc := range hostAddrs {
			addrBits := 128
			if ipc.IP.To4() != nil {
				addrBits = 32
			}

			err := netlink.RouteAdd(&netlink.Route{
				LinkIndex: contVeth.Index,
				Scope:     netlink.SCOPE_LINK,
				Dst: &net.IPNet{
					IP:   ipc.IP,
					Mask: net.CIDRMask(addrBits, addrBits),
				},
			})

			if err != nil {
				return fmt.Errorf("failed to add host route dst %v: %v", ipc.IP, err)
			}
		}

		// add a default gateway pointed at the first hostAddr
		err = netlink.RouteAdd(&netlink.Route{
			LinkIndex: contVeth.Index,
			Scope:     netlink.SCOPE_UNIVERSE,
			Dst:       nil,
			Gw:        hostAddrs[0].IP,
		})
		if err != nil {
			return fmt.Errorf("failed to add default route %v: %v", hostAddrs[0].IP, err)
		}

		// Send a gratuitous arp for all borrowed v4 addresses
		for _, ipc := range pr.IPs {
			if ipc.Version == "4" {
				_ = arping.GratuitousArpOverIface(ipc.Address.IP, *contVeth)
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return hostInterface, containerInterface, nil
}

func setupHostVeth(vethName string, hostAddrs []netlink.Addr, result *current.Result) error {
	// hostVeth moved namespaces and may have a new ifindex
	//	veth, err := netlink.LinkByName(vethName)
	veth, err := net.InterfaceByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", vethName, err)
	}

	for _, ipc := range result.IPs {
		addrBits := 128
		if ipc.Address.IP.To4() != nil {
			addrBits = 32
		}

		err := netlink.RouteAdd(&netlink.Route{
			LinkIndex: veth.Index,
			Scope:     netlink.SCOPE_LINK,
			Dst: &net.IPNet{
				IP:   ipc.Address.IP,
				Mask: net.CIDRMask(addrBits, addrBits),
			},
		})

		if err != nil {
			return fmt.Errorf("failed to add host route dst %v: %v", ipc.Address.IP, err)
		}
	}

	// Send a gratuitous arp for all borrowed v4 addresses
	for _, ipc := range hostAddrs {
		if ipc.IP.To4() != nil {
			_ = arping.GratuitousArpOverIface(ipc.IP, *veth)
		}
	}

	return nil
}

// cmdAdd is called for ADD requests
func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	if conf.PrevResult == nil {
		return fmt.Errorf("must be called as chained plugin")
	}

	// This is some sample code to generate the list of container-side IPs.
	// We're casting the prevResult to a 0.3.0 response, which can also include
	// host-side IPs (but doesn't when converted from a 0.2.0 response).
	containerIPs := make([]net.IP, 0, len(conf.PrevResult.IPs))
	if conf.CNIVersion != "0.3.0" {
		for _, ip := range conf.PrevResult.IPs {
			containerIPs = append(containerIPs, ip.Address.IP)
		}
	} else {
		for _, ip := range conf.PrevResult.IPs {
			if ip.Interface == nil {
				continue
			}
			intIdx := *ip.Interface
			// Every IP is indexed in to the interfaces array, with "-1" standing
			// for an unknown interface (which we'll assume to be Container-side
			// Skip all IPs we know belong to an interface with the wrong name.
			if intIdx >= 0 && intIdx < len(conf.PrevResult.Interfaces) && conf.PrevResult.Interfaces[intIdx].Name != args.IfName {
				continue
			}
			containerIPs = append(containerIPs, ip.Address.IP)
		}
	}
	if len(containerIPs) == 0 {
		return fmt.Errorf("got no container IPs")
	}

	iface, err := netlink.LinkByName(conf.HostInterface)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", conf.HostInterface, err)
	}

	hostAddrs, err := netlink.AddrList(iface, netlink.FAMILY_ALL)
	if err != nil || len(hostAddrs) == 0 {
		return fmt.Errorf("failed to get host IP addresses for %q: %v", iface, err)
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	hostInterface, _, err := setupContainerVeth(netns, conf.ContainerInterface, conf.MTU, hostAddrs, conf.PrevResult)
	if err != nil {
		return err
	}

	if err = setupHostVeth(hostInterface.Name, hostAddrs, conf.PrevResult); err != nil {
		return err
	}

	if conf.IPMasq {
		ipv4 := false
		ipv6 := false
		// only enable forwarding if masq is enabled
		for _, ipc := range containerIPs {
			if ipc.To4() != nil {
				ipv4 = true
			} else {
				ipv6 = true
			}
		}
		if ipv4 {
			err := ip.EnableIP4Forward()
			if err != nil {
				return fmt.Errorf("Could not enable IPv6 forwarding: %v", err)
			}
		}
		if ipv6 {
			err := ip.EnableIP6Forward()
			if err != nil {
				return fmt.Errorf("Could not enable IPv6 forwarding: %v", err)
			}
		}

		chain := utils.FormatChainName(conf.Name, args.ContainerID)
		comment := utils.FormatComment(conf.Name, args.ContainerID)
		for _, ipc := range containerIPs {
			addrBits := 128
			if ipc.To4() != nil {
				addrBits = 32
			}

			if err = ip.SetupIPMasq(&net.IPNet{IP: ipc, Mask: net.CIDRMask(addrBits, addrBits)}, chain, comment); err != nil {
				return err
			}
		}
	}

	// Pass through the result for the next plugin
	return types.PrintResult(conf.PrevResult, conf.CNIVersion)
}

// cmdDel is called for DELETE requests
func cmdDel(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	if args.Netns == "" {
		return nil
	}

	// There is a netns so try to clean up. Delete can be called multiple times
	// so don't return an error if the device is already removed.
	// If the device isn't there then don't try to clean up IP masq either.
	var ipnets []netlink.Addr
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		var err error

		// lookup pod IPs from the args.IfName device (usually eth0)
		if conf.IPMasq {
			iface, err := netlink.LinkByName(args.IfName)
			if err != nil {
				if err.Error() == "Link not found" {
					return ip.ErrLinkNotFound
				}
				return fmt.Errorf("failed to lookup %q: %v", args.IfName, err)
			}

			ipnets, err = netlink.AddrList(iface, netlink.FAMILY_ALL)
			if err != nil || len(ipnets) == 0 {
				return fmt.Errorf("failed to get IP addresses for %q: %v", args.IfName, err)
			}
		}

		// rm the veth device
		err = ip.DelLinkByName(conf.ContainerInterface)
		if err != nil && err != ip.ErrLinkNotFound {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "ptp container interface teardown error: %v", err)
	}

	if conf.IPMasq {
		chain := utils.FormatChainName(conf.Name, args.ContainerID)
		comment := utils.FormatComment(conf.Name, args.ContainerID)
		for _, ipn := range ipnets {
			addrBits := 128
			if ipn.IP.To4() != nil {
				addrBits = 32
			}

			err = ip.TeardownIPMasq(&net.IPNet{IP: ipn.IP, Mask: net.CIDRMask(addrBits, addrBits)}, chain, comment)
		}
	}

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}
