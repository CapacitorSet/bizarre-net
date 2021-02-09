package main

import (
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/udp"
	"log"
	"net"
)

// Maps the in-tunnel source IP of the host to its UDP source
var clientUdpAddr map[string]net.Addr

func main() {
	config, md, err := bizarre.ReadConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	iface, err := bizarre.CreateInterface(config.TUN)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s up with IP %s.\n", iface.Name, iface.IPNet.String())
	defer iface.Close()

	udpSrv, err := udp.Transport{}.Listen(config, md)
	if err != nil {
		log.Fatal(err)
	}
	defer udpSrv.Close()

	clientUdpAddr = make(map[string]net.Addr)
	serverDoneChan := make(chan error, 1)

	go tunLoop(udpSrv, iface)
	go udpLoop(udpSrv, serverDoneChan, iface)

	select {
	case err = <-serverDoneChan:
		return
	}
}

func udpLoop(udpSrv net.PacketConn, serverDoneChan chan error, iface bizarre.Interface) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, udpSrc, err := udpSrv.ReadFrom(buffer)
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}

		pkt, isIPv6 := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		// Inspect the source address so packet responses (syn-akcs, etc) can be sent to the host
		netFlow := pkt.NetworkLayer().NetworkFlow()
		tunnelSrc, _ := netFlow.Endpoints()
		clientUdpAddr[tunnelSrc.String()] = udpSrc

		fmt.Printf("\nnet > bytes=%d from=%s\n", n, udpSrc.String())
		bizarre.PrintPacket(pkt, isIPv6)

		_, err = iface.Write(buffer[:n])
		if err != nil {
			log.Print("sendto: ", err)
			serverDoneChan <- err
			break
		}

		fmt.Printf("> %s bytes=%d to=%s\n", iface.Name, n, udpSrc.String())
	}
}

func tunLoop(udpSrv net.PacketConn, iface bizarre.Interface) {
	buffer := make([]byte, 4096)
	for {
		n, err := iface.Read(buffer)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}
		fmt.Printf("\n%s > bytes=%d\n", iface.Name, n)
		pkt, isIPv6 := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		bizarre.PrintPacket(pkt, isIPv6)
		if isIPv6 {
			log.Println("Skipping IPv6 pkt")
			continue
		}
		netFlow := pkt.NetworkLayer().NetworkFlow()
		_, tunnelDst := netFlow.Endpoints()
		udpAddr := clientUdpAddr[tunnelDst.String()]
		if udpAddr == nil {
			fmt.Print("No client UDP address found, skipping\n")
			continue
		}

		_, err = udpSrv.WriteTo(buffer[:n], udpAddr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("> net bytes=%d\n", n)
	}
}
