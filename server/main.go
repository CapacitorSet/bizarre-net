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

// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
var clientAddr map[string]net.Addr

// Maps the in-tunnel source IP of the host to its connection (used in Write for stream transports)
var clientConn map[string]net.Conn

// The TUN interface
var iface bizarre.Interface

func main() {
	config, md, err := bizarre.ReadConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	iface, err = bizarre.CreateInterface(config.TUN)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s up with IP %s.\n", iface.Name, iface.IPNet.String())
	defer iface.Close()

	genericTransport, err := getTransport(config)
	if err != nil {
		log.Fatal(err)
	}

	serverDoneChan := make(chan error)
	switch transport := genericTransport.(type) {
	case bizarre.StreamTransport:
		clientConn = make(map[string]net.Conn)
		server, err := transport.Listen(config, md)
		if err != nil {
			log.Fatal(err)
		}
		defer server.Close()
		go streamLoop(server, serverDoneChan)
		go tunStreamLoop(iface)
	case bizarre.DatagramTransport:
		clientAddr = make(map[string]net.Addr)
		server, err := transport.Listen(config, md)
		if err != nil {
			log.Fatal(err)
		}
		defer server.Close()
		go datagramLoop(server, serverDoneChan)
		go tunDatagramLoop(server, iface)
	default:
		log.Fatalf("transport %T implements neither StreamTransport nor DatagramTransport", transport)
	}

	select {
	case err = <-serverDoneChan:
		return
	}
}

func processNetPkt(packet []byte, remoteAddr net.Addr, registerClient func(string)) error {
	pkt, isIPv6 := bizarre.TryParse(packet)
	if pkt == nil {
		log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
		return nil
	}
	// Inspect the source address so packet responses (syn-acks, etc) can be sent to the host
	netFlow := pkt.NetworkLayer().NetworkFlow()
	tunnelSrc, _ := netFlow.Endpoints()
	registerClient(tunnelSrc.String())

	fmt.Printf("\nnet > bytes=%d from=%s\n", len(packet), remoteAddr.String())
	bizarre.PrintPacket(pkt, isIPv6)

	_, err := iface.Write(packet)
	if err != nil {
		log.Print("sendto: ", err)
		return err
	}
	fmt.Printf("> %s bytes=%d to=%s\n", iface.Name, len(packet), remoteAddr.String())
	return nil
}
