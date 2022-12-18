package sources

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
)

type TUNConfig struct {
	Name         string // The name of the network interface
	IP           string // The address and netmask in CIDR notation, eg. "10.0.0.1/24"
	DefaultRoute bool
}

type TUNSource struct {
	TUN        *water.Interface
	Name       string
	*net.IPNet // IP and netmask
}

func (S *TUNSource) Start(ch chan []byte) {
	buffer := make([]byte, 4096)
	for {
		n, err := S.TUN.Read(buffer)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}
		ch <- buffer[:n]
	}
}

func (S *TUNSource) Write(buf []byte) error {
	_, err := S.TUN.Write(buf)
	return err
}

var (
	_ Source = (*TUNSource)(nil) // Ensure that interface fields are implemented

	ioctlLock = &sync.Mutex{}
)


// CreateTUN creates a TUN with the given config, if Name != "".
func CreateTUN(config TUNConfig) (TUNSource, error) {
	if config.Name == "" {
		return TUNSource{}, nil
	}

	ioctlLock.Lock()
	defer ioctlLock.Unlock()

	ifaces, err := net.Interfaces()
	if err != nil {
		return TUNSource{}, err
	}
	for _, iface := range ifaces {
		if iface.Name == config.Name {
			return TUNSource{}, fmt.Errorf("an interface with this name already exists")
		}
	}

	tunConfig := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: config.Name,
		},
	}
	tun, err := water.New(tunConfig)
	if err != nil {
		return TUNSource{}, fmt.Errorf("creating TUN: %w", err)
	}

	link, err := tenus.NewLinkFrom(config.Name)
	if err != nil {
		return TUNSource{}, fmt.Errorf("reading TUN: %w", err)
	}
	ip, subnet, err := net.ParseCIDR(config.IP)
	if err != nil {
		return TUNSource{}, fmt.Errorf("parsing TUN subnet: %w", err)
	}
	subnet.IP = ip
	err = link.SetLinkIp(ip, subnet)
	if err != nil {
		return TUNSource{}, fmt.Errorf("configuring TUN: %w", err)
	}
	err = link.SetLinkUp()
	if err != nil {
		return TUNSource{}, fmt.Errorf("configuring TUN: %w", err)
	}

	if config.DefaultRoute {
		err = link.SetLinkDefaultGw(&ip)
		if err != nil {
			return TUNSource{}, fmt.Errorf("creating default route: %w", err)
		}
	}

	return TUNSource{
		TUN:   tun,
		Name:  config.Name,
		IPNet: subnet,
	}, nil
}
