package main

import (
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

func main() {
	config, err := ReadConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	iface, err := bizarre.CreateInterface(bizarre.InterfaceConfig{
		Prefix:  config.TunPrefix,
		Address: config.TunIP,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s up.\n", iface.Name)
	defer iface.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", config.ServerIP)
	if err != nil {
		log.Fatal(err)
	}
	udpConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal(err)
	}

	serverDoneChan := make(chan error, 1)

	go tunLoop(udpConn, iface, serverDoneChan)
	go udpLoop(udpConn, iface, serverDoneChan)

	for {
		select {
		case <-serverDoneChan:
			return
		}
	}
}

func udpLoop(udpConn *net.UDPConn, iface bizarre.Interface, serverDoneChan chan error) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, addr, err := udpConn.ReadFrom(buffer)
		if err != nil {
			log.Println("udpLoop: " + err.Error())
			serverDoneChan <- err
			break
		}

		fmt.Printf("\nnet > bytes=%d from=%s\n", n, addr.String())
		pkt, isIPv6 := bizarre.TryParse(buffer[:n])
		bizarre.PrintPacket(pkt, isIPv6)

		_, err = iface.Write(buffer[:n])
		if err != nil {
			log.Println("udpLoop: " + err.Error())
			serverDoneChan <- err
			break
		}

		fmt.Printf("> %s bytes=%d\n", iface.Name, n)
	}
}

func tunLoop(udpConn *net.UDPConn, iface bizarre.Interface, serverDoneChan chan error) {
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

		udpConn.Write(buffer[:n])
		fmt.Printf("net > bytes=%d\n", n)
	}
}