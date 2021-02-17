package client

import (
	"errors"
	"fmt"
	"github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/cat"
	"github.com/CapacitorSet/bizarre-net/socket"
	"github.com/CapacitorSet/bizarre-net/udp"
	"log"
	"net"
	"strings"
	"sync"
)

func getTransport(config bizarre_net.Config) (bizarre_net.Transport, error) {
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
	config, md, err := bizarre_net.ReadConfig(configPath)
	if err != nil {
		return err
	}
	iface, err := bizarre_net.CreateInterface(config.TUN, ioctlLock)
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

func transportLoop(client net.Conn, iface bizarre_net.Interface, serverDoneChan chan error) {
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

		pkt, isIPv6 := bizarre_net.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if bizarre_net.IsChatter(pkt) {
			continue
		}
		if isIPv6 {
			fmt.Println("Skipping IPv6 pkt")
			continue
		}
		fmt.Printf("\nnet > bytes=%d\n", n)
		bizarre_net.PrintPacket(pkt, isIPv6)

		// todo: handle iface write fails gracefully if not an IP packet (buffer[0] & 0xF0 != 0x4, 0x6)
		_, err = iface.Write(buffer[:n])
		if err != nil {
			log.Println("transportLoop: " + err.Error())
			serverDoneChan <- err
			break
		}

		fmt.Printf("> %s bytes=%d\n", iface.Name, n)
	}
}

func tunLoop(client net.Conn, iface bizarre_net.Interface, serverDoneChan chan error) {
	buffer := make([]byte, 4096)
	for {
		n, err := iface.Read(buffer)
		if err != nil {
			log.Println("tunLoop: " + err.Error())
			serverDoneChan <- err
			break
		}

		pkt, isIPv6 := bizarre_net.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if bizarre_net.IsChatter(pkt) {
			continue
		}
		if isIPv6 {
			fmt.Println("Skipping IPv6 pkt")
			continue
		}
		fmt.Printf("\n%s > bytes=%d\n", iface.Name, n)

		client.Write(buffer[:n])
		fmt.Printf("net > bytes=%d\n", n)
	}
}
