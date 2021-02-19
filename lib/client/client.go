package client

import (
	"errors"
	"fmt"
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

// configPath is the path to the TOML config
// ioctlLock is an optional mutex to lock ioctl (i.e. TUN creation) calls. It avoids crashes when launching eg. both
//   a client and a server at the same time in tests
func Run(configPath string, ioctlLock *sync.Mutex) error {
	config, md, err := bizarre.ReadConfig(configPath)
	if err != nil {
		return err
	}
	iface, err := bizarre.CreateInterface(config.TUN, ioctlLock)
	if err != nil {
		return err
	}
	log.Printf("%s up.\n", iface.Name)
	defer iface.Close()

	transport, err := getTransport(config)
	if err != nil {
		return err
	}
	client, err := transport.Dial(config, md)
	if err != nil {
		return err
	}

	serverDoneChan := make(chan error, 1)

	go tunLoop(client, iface, serverDoneChan)
	go transportLoop(client, iface, serverDoneChan)

	for {
		select {
		case err = <-serverDoneChan:
			return err
		}
	}
}

func transportLoop(client net.Conn, iface bizarre.Interface, serverDoneChan chan error) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, err := client.Read(buffer)
		if err != nil {
			log.Println("transportLoop: " + err.Error())
			serverDoneChan <- err
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
		_, err = iface.Write(buffer[:n])
		if err != nil {
			log.Println("transportLoop: " + err.Error())
			serverDoneChan <- err
			break
		}

		fmt.Printf("net=>tun: bytes=%d\n", n)
	}
}

func tunLoop(client net.Conn, iface bizarre.Interface, serverDoneChan chan error) {
	buffer := make([]byte, 4096)
	for {
		n, err := iface.Read(buffer)
		if err != nil {
			log.Println("tunLoop: " + err.Error())
			serverDoneChan <- err
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
		fmt.Printf("\ntun=>net: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		client.Write(buffer[:n])
		fmt.Printf("tun=>net: bytes=%d\n", n)
	}
}
