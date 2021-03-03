package server

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/cat"
	"github.com/CapacitorSet/bizarre-net/udp"
	"log"
	"net"
	"strings"
	"sync"
)

func getTransport(config bizarre.Config, md toml.MetaData) (bizarre.Transport, error) {
	switch strings.ToLower(config.Transport) {
	case "udp":
		return udp.NewTransport(config, md)
	case "cat":
		return cat.NewServerTransport(config, md)
	default:
		return nil, errors.New("no such transport: " + config.Transport)
	}
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
	log.Printf("%s up with IP %s.\n", iface.Name, iface.IPNet.String())

	genericTransport, err := getTransport(config, md)
	if err != nil {
		return nil, err
	}

	base := BaseServer{
		Interface: iface,
		Config:    config,
	}
	switch transport := genericTransport.(type) {
	case bizarre.StreamTransport:
		return StreamServer{
			BaseServer: base,
			Transport:  transport,
			clientConn: make(map[string]net.Conn),
		}, nil
	case bizarre.DatagramTransport:
		return DatagramServer{
			BaseServer: base,
			Transport:  transport,
			clientAddr: make(map[string]net.Addr),
		}, nil
	default:
		return nil, errors.New(fmt.Sprintf("transport %T implements neither StreamTransport nor DatagramTransport", transport))
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
