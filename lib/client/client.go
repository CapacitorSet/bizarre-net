package client

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

func getTransport(config bizarre.Config, md toml.MetaData) (bizarre.Transport, error) {
	switch strings.ToLower(config.Transport) {
	case "udp":
		return udp.NewTransport(config, md)
	case "cat":
		return cat.NewClientTransport(config, md)
	case "socket":
		return socket.NewTransport(config, md)
	default:
		return nil, errors.New("no such transport: " + config.Transport)
	}
}

type Client struct {
	bizarre.Interface
	bizarre.Config
	md toml.MetaData
	bizarre.Transport
}

// configPath is the path to the TOML config
// ioctlLock is an optional mutex to lock ioctl (i.e. TUN creation) calls. It avoids crashes when launching eg. both
//   a client and a server at the same time in tests
func NewClient(configPath string, ioctlLock *sync.Mutex) (Client, error) {
	config, md, err := bizarre.ReadConfig(configPath)
	if err != nil {
		return Client{}, err
	}
	iface, err := bizarre.CreateInterface(config.TUN, ioctlLock)
	if err != nil {
		return Client{}, err
	}
	log.Printf("%s up.\n", iface.Name)

	transport, err := getTransport(config, md)
	if err != nil {
		return Client{}, err
	}
	return Client{
		Interface: iface,
		Config:    config,
		md:        md,
		Transport: transport,
	}, nil
}

func (C Client) Run() error {
	if C.Config.TUN.SetDefaultGW {
		log.Println("Routing all traffic through " + C.Interface.Name)
		err := bizarre.SetDefaultGateway(C.Interface)
		if err != nil {
			return err
		}
		hasConflict, err := C.Transport.HasIPRoutingConflict(C.Interface)
		if err != nil {
			panic(err)
			return err
		}
		if hasConflict {
			if C.Config.SkipRoutingCheck {
				log.Println("The IP endpoint is routed through the tunnel. Ignoring due to SkipRoutingCheck=true.")
			} else {
				log.Fatalln("The IP/host you are trying to use as an endpoint seems to be routed through the tunnel itself; this likely won't work. Review the routing table, or add SkipRoutingCheck=true in the config if you know what you're doing.")
			}
		}
	}

	// Todo: check that IP transports (udp etc) are not routed through the interface
	client, err := C.Transport.Dial()
	if err != nil {
		return err
	}

	doneChan := make(chan error, 1)

	go C.tunLoop(client, doneChan)
	go C.transportLoop(client, doneChan)

	for {
		select {
		case err = <-doneChan:
			return err
		}
	}
}

func (C Client) tunLoop(client net.Conn, doneChan chan error) {
	buffer := make([]byte, 4096)
	for {
		n, err := C.Interface.Read(buffer)
		if err != nil {
			log.Println("tunLoop: " + err.Error())
			doneChan <- err
			break
		}

		pkt := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if C.Config.DropChatter && bizarre.IsChatter(pkt) {
			continue
		}
		fmt.Printf("\ntun=>net: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		client.Write(buffer[:n])
		fmt.Printf("tun=>net: bytes=%d\n", n)
	}
}

func (C Client) transportLoop(client net.Conn, doneChan chan error) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, err := client.Read(buffer)
		if err != nil {
			log.Println("transportLoop: " + err.Error())
			doneChan <- err
			break
		}

		pkt := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if bizarre.IsChatter(pkt) {
			continue
		}
		fmt.Printf("\nnet=>tun: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		// todo: handle iface write fails gracefully if not an IP packet (buffer[0] & 0xF0 != 0x4, 0x6)
		_, err = C.Interface.Write(buffer[:n])
		if err != nil {
			log.Println("transportLoop: " + err.Error())
			doneChan <- err
			break
		}

		fmt.Printf("net=>tun: bytes=%d\n", n)
	}
}
