module github.com/lyft/cni-ipvlan-vpc-k8s

go 1.13

require (
	github.com/Microsoft/go-winio v0.4.11
	github.com/aws/aws-sdk-go v1.12.79
	github.com/containernetworking/cni v0.6.0
	github.com/containernetworking/plugins v0.7.4
	github.com/coreos/go-iptables v0.4.0
	github.com/docker/distribution v2.6.2+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.3.3
	github.com/go-ini/ini v1.39.0
	github.com/j-keck/arping v0.0.0-20160618110441-2cf9dc699c56
	github.com/jmespath/go-jmespath v0.0.0-20160202185014-0b12d6b521d8
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/pkg/errors v0.8.0
	github.com/urfave/cli v1.20.0
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc
	golang.org/x/net v0.0.0-20181114220301-adae6a3d119a
	golang.org/x/sys v0.0.0-20181119195503-ec83556a53fe
)
