package main

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

func main() {
	config, md, err := bizarre.ReadConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	iface, err := bizarre.CreateInterface(config.TUN)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s up.\n", iface.Name)
	defer iface.Close()

	transport, err := getTransport(config)
	if err != nil {
		log.Fatal(err)
	}
	client, err := transport.Dial(config, md)
	if err != nil {
		log.Fatal(err)
	}

	serverDoneChan := make(chan error, 1)

	go tunLoop(client, iface, serverDoneChan)
	go transportLoop(client, iface, serverDoneChan)

	for {
		select {
		case <-serverDoneChan:
			return
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

		fmt.Printf("\nnet > bytes=%d\n", n)
		pkt, isIPv6 := bizarre.TryParse(buffer[:n])
		if pkt != nil {
			bizarre.PrintPacket(pkt, isIPv6)
		} else {
			fmt.Println("Unknown packet.")
		}

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

func tunLoop(client net.Conn, iface bizarre.Interface, serverDoneChan chan error) {
	buffer := make([]byte, 4096)
	for {
		n, err := iface.Read(buffer)
		if err != nil {
			log.Println("tunLoop: " + err.Error())
			serverDoneChan <- err
			break
		}
		fmt.Printf("\n%s > bytes=%d\n", iface.Name, n)
		pkt, isIPv6 := bizarre.TryParse(buffer[:n])
		bizarre.PrintPacket(pkt, isIPv6)
		if isIPv6 {
			fmt.Println("Skipping IPv6 pkt")
			continue
		}

		client.Write(buffer[:n])
		fmt.Printf("net > bytes=%d\n", n)
	}
}
