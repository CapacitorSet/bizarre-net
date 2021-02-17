package server

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/cat"
	"github.com/CapacitorSet/bizarre-net/socket"
	"github.com/CapacitorSet/bizarre-net/udp"
	"log"
	"net"
	"strings"
	"sync"
)

func getTransport(config bizarre.Config) (bizarre.Transport, error) {
	switch strings.ToLower(config.Transport) {
	case "udp":
		return udp.Transport{}, nil
	case "cat":
		return cat.Transport{}, nil
	case "socket":
		return socket.Transport{}, nil
	default:
		return nil, errors.New("no such transport: " + config.Transport)
	}
}

type Server struct {
	// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
	clientAddr map[string]net.Addr

	// Maps the in-tunnel source IP of the host to its connection (used in Write for stream transports)
	clientConn map[string]net.Conn

	// The TUN interface
	iface bizarre.Interface

	// TOML config data
	config bizarre.Config
	md toml.MetaData

	transport bizarre.Transport
}

// ioctlLock is an optional mutex to lock ioctl (i.e. TUN creation) calls. It avoids crashes when launching eg. both
//   a client and a server at the same time in tests
func NewServer(configFile string, ioctlLock *sync.Mutex) (Server, error) {
	config, md, err := bizarre.ReadConfig(configFile)
	if err != nil {
		return Server{}, err
	}
	iface, err := bizarre.CreateInterface(config.TUN, ioctlLock)
	if err != nil {
		return Server{}, err
	}
	log.Printf("%s up with IP %s.\n", iface.Name, iface.IPNet.String())

	transport, err := getTransport(config)
	if err != nil {
		return Server{}, err
	}

	s := Server{
		iface: iface,
		config: config,
		md: md,
		transport: transport,
	}
	switch transport.(type) {
	case bizarre.StreamTransport:
		s.clientConn = make(map[string]net.Conn)
	case bizarre.DatagramTransport:
		s.clientAddr = make(map[string]net.Addr)
	default:
		return Server{}, errors.New(fmt.Sprintf("transport %T implements neither StreamTransport nor DatagramTransport", transport))
	}

	return s, nil
}


func (S Server) Run() error {
	serverDoneChan := make(chan error)
	switch transport := S.transport.(type) {
	case bizarre.StreamTransport:
		server, err := transport.Listen(S.config, S.md)
		if err != nil {
			return err
		}
		defer server.Close()
		go S.streamLoop(server, serverDoneChan)
		go S.tunStreamLoop(S.iface)
	case bizarre.DatagramTransport:
		server, err := transport.Listen(S.config, S.md)
		if err != nil {
			return err
		}
		defer server.Close()
		go S.datagramLoop(server, serverDoneChan)
		go S.tunDatagramLoop(server, S.iface)
	default:
	}

	select {
	case err := <-serverDoneChan:
		return err
	}
}

func (S Server) processNetPkt(packet []byte, remoteAddr net.Addr, registerClient func(string)) error {
	pkt, isIPv6 := bizarre.TryParse(packet)
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

	fmt.Printf("\nnet > bytes=%d from=%s\n", len(packet), remoteAddr.String())
	bizarre.PrintPacket(pkt, isIPv6)

	_, err := S.iface.Write(packet)
	if err != nil {
		log.Print("sendto: ", err)
		return err
	}
	fmt.Printf("> %s bytes=%d to=%s\n", S.iface.Name, len(packet), remoteAddr.String())
	return nil
}
