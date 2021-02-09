package bizarre_net

import (
	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
	"net"
	"strconv"
	"strings"
)

// Functions for setting up the network interface

type TUNConfig struct {
	Prefix string // The prefix of the network interface, eg. "bizarre" will create bizarre0, bizarre1...
	IP     string // The address and netmask in CIDR notation, eg. "10.0.0.1/24"
}

type Interface struct {
	*water.Interface // The TUN
	Name string
	*net.IPNet // IP and netmask
}

func findIfaceName(prefix string) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	// Find the highest N for which bizarreN doesn't exist
	bizarreIfaceNum := int64(0)
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, prefix) {
			ifaceNum, err := strconv.ParseInt(iface.Name[7:], 10, 16)
			if err != nil {
				continue
			}
			if ifaceNum >= bizarreIfaceNum {
				bizarreIfaceNum = ifaceNum + 1
			}
		}
	}
	return prefix + strconv.Itoa(int(bizarreIfaceNum)), nil
}

func CreateInterface(config TUNConfig) (Interface, error) {
	var iface Interface

	name, err := findIfaceName(config.Prefix)
	if err != nil {
		return Interface{}, err
	}
	iface.Name = name

	tunConfig := water.Config{
		DeviceType: water.TUN,
	}
	tunConfig.Name = iface.Name
	tun, err := water.New(tunConfig)
	if err != nil {
		return Interface{}, err
	}
	iface.Interface = tun

	link, err := tenus.NewLinkFrom(iface.Name)
	if err != nil {
		return Interface{}, err
	}
	ip, subnet, err := net.ParseCIDR(config.IP)
	if err != nil {
		return Interface{}, err
	}
	iface.IPNet = subnet
	err = link.SetLinkIp(ip, subnet)
	if err != nil {
		return Interface{}, err
	}
	err = link.SetLinkUp()
	if err != nil {
		return Interface{}, err
	}

	return iface, nil
}