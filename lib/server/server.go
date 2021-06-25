package server

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/transports/cat"
	"github.com/CapacitorSet/bizarre-net/transports/ping"
	"github.com/CapacitorSet/bizarre-net/transports/udp"
	"log"
	"net"
	"strings"
	"sync"
)

func getTransport(config bizarre.Config, md toml.MetaData) (bizarre.ServerTransport, error) {
	var packetServer bizarre.PacketServer
	var connServer bizarre.ConnServer
	var err error
	switch strings.ToLower(config.Transport) {
	case "udp":
		packetServer, err = udp.Server(config, md)
	case "ping":
		packetServer, err = ping.Server(config, md)
	case "cat":
		packetServer, err = cat.Server(config, md)
	default:
		err = errors.New("no such transport: " + config.Transport)
	}
	if err != nil {
		return bizarre.ServerTransport{}, err
	}
	return bizarre.ServerTransport{connServer, packetServer}, nil
}

type BaseServer struct {
	bizarre.Interface
	bizarre.Config
}
type Server interface {
	Run() error
}

// ioctlLock is an optional mutex to lock ioctl (i.e. TUN creation) calls. It avoids crashes when launching eg. both
//   a client and a server at the same time in tests
func NewServer(configFile string, ioctlLock *sync.Mutex) (Server, error) {
	config, md, err := bizarre.ReadConfig(configFile)
	if err != nil {
		return nil, err
	}
	iface, err := bizarre.CreateInterface(config.TUN, ioctlLock)
	if err != nil {
		return nil, err
	}
	log.Printf("New interface: %s with IP %s\n", iface.Name, iface.IP.String())

	log.Printf("Accepting connections via %s\n", config.Transport)
	genericTransport, err := getTransport(config, md)
	if err != nil {
		return nil, err
	}

	base := BaseServer{
		Interface: iface,
		Config:    config,
	}
	if transport := genericTransport.ConnServer; transport != nil {
		return ConnServer{
			BaseServer: base,
			Transport:  transport,
			clientConn: make(map[string]net.Conn),
		}, nil
	} else if transport := genericTransport.PacketServer; transport != nil {
		return PacketServer{
			BaseServer: base,
			Transport:  transport,
			clientAddr: make(map[string]net.Addr),
		}, nil
	} else {
		return nil, errors.New(fmt.Sprintf("transport %T is neither a ConnServer nor a PacketServer", transport))
	}
}

func (S BaseServer) processNetPkt(packet []byte, registerClient func(string)) error {
	pkt := bizarre.TryParse(packet)
	if pkt == nil {
		log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
		return nil
	}

	if bizarre.IsChatter(pkt) {
		return nil
	}

	// Inspect the source address so packet responses (syn-acks, etc) can be sent to the host
	netFlow := pkt.NetworkLayer().NetworkFlow()
	tunnelSrc, _ := netFlow.Endpoints()
	registerClient(tunnelSrc.String())

	fmt.Printf("\nnet=>tun: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), len(packet))

	_, err := S.Interface.Write(packet)
	if err != nil {
		log.Print("sendto: ", err)
		return err
	}
	fmt.Printf("net=>tun: bytes=%d\n", len(packet))
	return nil
}
