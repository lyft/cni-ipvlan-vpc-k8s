package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli"

	"github.com/lyft/cni-ipvlan-vpc-k8s/aws"
	"github.com/lyft/cni-ipvlan-vpc-k8s/lib"
	"github.com/lyft/cni-ipvlan-vpc-k8s/nl"
)

var version string

// Build a filter from input
func filterBuild(input string) (map[string]string, error) {
	if input == "" {
		return nil, nil
	}

	ret := make(map[string]string)
	tuples := strings.Split(input, ",")
	for _, t := range tuples {
		kv := strings.Split(t, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Invalid filter specified %v", t)
		}
		if len(kv[0]) <= 0 || len(kv[1]) <= 0 {
			return nil, fmt.Errorf("Zero length filter specified: %v", t)
		}

		ret[kv[0]] = kv[1]
	}

	return ret, nil
}

func actionNewInterface(c *cli.Context) error {
	return lib.LockfileRun(func() error {
		filtersRaw := c.String("subnet_filter")
		filters, err := filterBuild(filtersRaw)
		if err != nil {
			fmt.Printf("Invalid filter specification %v", err)
			return err
		}

		secGrps := c.Args()

		if len(secGrps) <= 0 {
			fmt.Println("please specify security groups")
			return fmt.Errorf("need security groups")
		}
		newIf, err := aws.DefaultClient.NewInterface(secGrps, filters)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println(newIf)
		return nil

	})
}

func actionBugs(c *cli.Context) error {
	return lib.LockfileRun(func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "bug\tafflicted\t")
		for _, bug := range aws.ListBugs(aws.DefaultClient) {
			fmt.Fprintf(w, "%s\t%v\t\n", bug.Name, bug.HasBug())
		}
		w.Flush()
		return nil
	})
}

func actionRemoveInterface(c *cli.Context) error {
	return lib.LockfileRun(func() error {
		interfaces := c.Args()

		if len(interfaces) <= 0 {
			fmt.Println("please specify an interface")
			return fmt.Errorf("Insufficent Arguments")
		}

		if err := aws.DefaultClient.RemoveInterface(interfaces); err != nil {
			fmt.Println(err)
			return err
		}

		return nil
	})
}

func actionDeallocate(c *cli.Context) error {
	return lib.LockfileRun(func() error {
		releaseIps := c.Args()
		for _, toRelease := range releaseIps {

			if len(toRelease) < 6 {
				fmt.Println("please specify an IP")
				return fmt.Errorf("Invalid IP")
			}

			ip := net.ParseIP(toRelease)
			if ip == nil {
				fmt.Println("please specify a valid IP")
				return fmt.Errorf("IP parse error")
			}

			err := aws.DefaultClient.DeallocateIP(&ip)
			if err != nil {
				fmt.Printf("deallocation failed: %v\n", err)
				return err
			}
		}
		return nil
	})
}

func actionAllocate(c *cli.Context) error {
	return lib.LockfileRun(func() error {
		index := c.Int("index")
		res, err := aws.DefaultClient.AllocateIPFirstAvailableAtIndex(index)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fmt.Printf("allocated %v on %v\n", res.IP, res.Interface.LocalName())
		return nil

	})
}

func actionFreeIps(c *cli.Context) error {
	ips, err := aws.FindFreeIPsAtIndex(0, false)
	if err != nil {
		fmt.Println(err)
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "adapter\tip\t")
	for _, ip := range ips {
		fmt.Fprintf(w, "%v\t%v\t\n",
			ip.Interface.LocalName(),
			ip.IP)
	}
	w.Flush()
	return nil
}

func actionLimits(c *cli.Context) error {
	limit := aws.DefaultClient.ENILimits()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "adapters\tipv4\tipv6\t")
	fmt.Fprintf(w, "%v\t%v\t%v\t\n", limit.Adapters,
		limit.IPv4,
		limit.IPv6)
	w.Flush()
	return nil
}

func actionMaxPods(c *cli.Context) error {
	limit := aws.DefaultClient.ENILimits()
	fmt.Printf("%d\n", (limit.Adapters-1)*limit.IPv4)
	return nil
}

func actionAddr(c *cli.Context) error {
	ips, err := nl.GetIPs()
	if err != nil {
		fmt.Println(err)
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "iface\tip\t")
	for _, ip := range ips {
		fmt.Fprintf(w, "%v\t%v\t\n",
			ip.Label,
			ip.IPNet.IP)
	}
	w.Flush()

	return nil
}

func actionEniIf(c *cli.Context) error {
	interfaces, err := aws.DefaultClient.GetInterfaces()
	if err != nil {
		fmt.Println(err)
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "iface\tmac\tid\tsubnet\tsubnet_cidr\tsecgrps\tvpc\tips\t")
	for _, iface := range interfaces {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t\n", iface.LocalName(),
			iface.Mac,
			iface.ID,
			iface.SubnetID,
			iface.SubnetCidr,
			iface.SecurityGroupIds,
			iface.VpcID,
			iface.IPv4s)

	}

	w.Flush()
	return nil
}

func actionVpcCidr(c *cli.Context) error {
	interfaces, err := aws.DefaultClient.GetInterfaces()
	if err != nil {
		fmt.Println(err)
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "iface\tmetadata cidr\taws api cidr\t")
	for _, iface := range interfaces {
		apiCidrs, _ := aws.DefaultClient.DescribeVPCCIDRs(iface.VpcID)

		fmt.Fprintf(w, "%s\t%v\t%v\t\n",
			iface.LocalName(),
			iface.VpcCidrs,
			apiCidrs)
	}
	w.Flush()
	return nil
}

func actionVpcPeerCidr(c *cli.Context) error {
	interfaces, err := aws.DefaultClient.GetInterfaces()
	if err != nil {
		fmt.Println(err)
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "iface\tpeer_dcidr\t")
	for _, iface := range interfaces {
		apiCidrs, _ := aws.DefaultClient.DescribeVPCPeerCIDRs(iface.VpcID)

		fmt.Fprintf(w, "%s\t%v\t\n",
			iface.LocalName(),
			apiCidrs)
	}
	w.Flush()
	return nil
}

func actionSubnets(c *cli.Context) error {
	subnets, err := aws.DefaultClient.GetSubnetsForInstance()
	if err != nil {
		fmt.Println(err)
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "id\tcidr\tdefault\taddresses_available\ttags\t")
	for _, subnet := range subnets {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t\n",
			subnet.ID,
			subnet.Cidr,
			subnet.IsDefault,
			subnet.AvailableAddressCount,
			subnet.Tags)
	}

	w.Flush()

	return nil
}

func actionRegistryList(c *cli.Context) error {
	return lib.LockfileRun(func() error {

		reg := &aws.Registry{}
		ips, err := reg.List()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ip\t")
		for _, ip := range ips {
			fmt.Fprintf(w, "%v\t\n",
				ip)
		}
		w.Flush()
		return nil
	})
}

func actionRegistryGc(c *cli.Context) error {
	return lib.LockfileRun(func() error {

		reg := &aws.Registry{}
		freeAfter := c.Duration("free-after")
		if freeAfter <= 0*time.Second {
			fmt.Fprintf(os.Stderr,
				"Invalid duration specified. free-after must be > 0 seconds. Got %v. Please specify with --free-after=[time]\n", freeAfter)
			return fmt.Errorf("invalid duration")
		}

		// Insert free-after jitter of 15% of the period
		freeAfter = aws.Jitter(freeAfter, 0.15)

		// Invert free-after
		freeAfter *= -1

		ips, err := reg.TrackedBefore(time.Now().Add(freeAfter))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}

		// grab a list of in-use IPs to sanity check
		assigned, err := nl.GetIPs()
		if err != nil {
			return err
		}

	OUTER:
		for _, ip := range ips {
			// forget IPs that are actually in use and skip over
			for _, assignedIP := range assigned {
				if assignedIP.IPNet.IP.Equal(ip) {
					reg.ForgetIP(ip)
					continue OUTER
				}
			}
			err := aws.DefaultClient.DeallocateIP(&ip)
			if err == nil {
				reg.ForgetIP(ip)
			} else {
				fmt.Fprintf(os.Stderr, "Can't deallocate %v due to %v", ip, err)
			}
		}

		return nil
	})
}

func main() {
	if !aws.DefaultClient.Available() {
		fmt.Fprintln(os.Stderr, "This command must be run from a running ec2 instance")
		os.Exit(1)
	}

	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "This command must be run as root")
		os.Exit(1)
	}

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:      "new-interface",
			Usage:     "Create a new interface",
			Action:    actionNewInterface,
			ArgsUsage: "[--subnet_filter=k,v] [security_group_ids...]",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "subnet_filter",
					Usage: "Comma separated key=value filters to restrict subnets",
				},
			},
		},
		{
			Name:      "remove-interface",
			Usage:     "Remove an existing interface",
			Action:    actionRemoveInterface,
			ArgsUsage: "[interface_id...]",
		},
		{
			Name:      "deallocate",
			Usage:     "Deallocate a private IP",
			Action:    actionDeallocate,
			ArgsUsage: "[ip...]",
		},
		{
			Name:   "allocate-first-available",
			Usage:  "Allocate a private IP on the first available interface",
			Action: actionAllocate,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "index"},
			},
		},
		{
			Name:   "free-ips",
			Usage:  "List all currently unassigned AWS IP addresses",
			Action: actionFreeIps,
		},
		{
			Name:   "eniif",
			Usage:  "List all ENI interfaces and their setup with addresses",
			Action: actionEniIf,
		},
		{
			Name:   "addr",
			Usage:  "List all bound IP addresses",
			Action: actionAddr,
		},
		{
			Name:   "subnets",
			Usage:  "Show available subnets for this host",
			Action: actionSubnets,
		},
		{
			Name:   "limits",
			Usage:  "Display limits for ENI for this instance type",
			Action: actionLimits,
		},
		{
			Name:   "maxpods",
			Usage:  "Return a single number specifying the maximum number of pod addresses that can be used on this instance",
			Action: actionMaxPods,
		},
		{
			Name:   "bugs",
			Usage:  "Show any bugs associated with this instance",
			Action: actionBugs,
		},
		{
			Name:   "vpccidr",
			Usage:  "Show the VPC CIDRs associated with current interfaces",
			Action: actionVpcCidr,
		},
		{
			Name:   "vpcpeercidr",
			Usage:  "Show the peered VPC CIDRs associated with current interfaces",
			Action: actionVpcPeerCidr,
		},
		{
			Name:   "registry-list",
			Usage:  "List all known free IPs in the internal registry",
			Action: actionRegistryList,
		},
		{
			Name:   "registry-gc",
			Usage:  "Free all IPs that have remained unused for a given time interval",
			Action: actionRegistryGc,
			Flags: []cli.Flag{
				cli.DurationFlag{Name: "free-after",
					Value: 0 * time.Second},
			},
		},
	}
	app.Version = version
	app.Copyright = "(c) 2017-2018 Lyft Inc."
	app.Usage = "Interface with ENI adapters and CNI bindings for those"
	app.Run(os.Args)
}
