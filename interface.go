package bizarre_net

import (
	"github.com/google/gopacket/routing"
	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Functions for setting up the network interface

type TUNConfig struct {
	Prefix       string // The prefix of the network interface, eg. "bizarre" will create bizarre0, bizarre1...
	IP           string // The address and netmask in CIDR notation, eg. "10.0.0.1/24"
	SetDefaultGW bool   // Use this interface as default route?
}

type Interface struct {
	*water.Interface // The TUN
	Name             string
	*net.IPNet       // IP and netmask
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
			ifaceNum, err := strconv.ParseInt(iface.Name[len(prefix):], 10, 16)
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

// ioctlLock is an optional mutex to lock ioctl (i.e. TUN creation) calls. It avoids crashes when launching eg. both
//   a client and a server at the same time in tests
func CreateInterface(config TUNConfig, ioctlLock *sync.Mutex) (Interface, error) {
	var iface Interface

	if ioctlLock != nil {
		ioctlLock.Lock()
		defer ioctlLock.Unlock()
	}

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
	iface.IPNet.IP = ip
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

func SetDefaultGateway(i Interface) error {
	link, err := tenus.NewLinkFrom(i.Name)
	if err != nil {
		return err
	}
	err = link.SetLinkDefaultGw(&i.IP)
	if err != nil {
		return err
	}
	return nil
}

// Is the given IP routed through this interface?
// Used to detect when the transport endpoint would be incorrectly tunneled in IP transports like UDP
func (I Interface) IsRoutedThrough(ip net.IP) (bool, error) {
	router, err := routing.New()
	if err != nil {
		return false, err
	}
	routeIface, _, _, err := router.Route(ip)
	if err != nil {
		return false, err
	}
	return routeIface.Name == I.Name, nil
}
