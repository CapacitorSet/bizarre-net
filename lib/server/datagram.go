package server

import (
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

// Handles packets from a datagram transport or from a TCP-like connection
func (S Server) datagramLoop(conn net.PacketConn, serverDoneChan chan error) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, transportSrc, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}

		err = S.processNetPkt(buffer[:n], transportSrc, func(tunnelSrc string) {
			S.clientAddr[tunnelSrc] = transportSrc
		})
		if err != nil {
			serverDoneChan <- err
			break
		}

		_, err = S.iface.Write(buffer[:n])
		if err != nil {
			log.Print("sendto: ", err)
			serverDoneChan <- err
			break
		}
	}
}

func (S Server) tunDatagramLoop(server net.PacketConn, iface bizarre.Interface) {
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
		addr := S.clientAddr[tunnelDst.String()]
		if addr == nil {
			fmt.Print("No client addr found, skipping\n")
			continue
		}

		_, err = server.WriteTo(buffer[:n], addr)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		fmt.Printf("> net bytes=%d\n", n)
	}
}
