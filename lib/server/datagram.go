package server

import (
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

type DatagramServer struct {
	BaseServer
	Transport bizarre.DatagramTransport

	// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
	clientAddr map[string]net.Addr
}

func (D DatagramServer) Run() error {
	serverDoneChan := make(chan error)
	server, err := D.Transport.Listen(D.config, D.md)
	if err != nil {
		return err
	}
	defer server.Close()
	go D.serverLoop(server, serverDoneChan)
	go D.tunLoop(server)

	select {
	case err := <-serverDoneChan:
		return err
	}
}

// Handles packets from a datagram transport or from a TCP-like connection
func (D DatagramServer) serverLoop(conn net.PacketConn, serverDoneChan chan error) {
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

		err = D.processNetPkt(buffer[:n], func(tunnelSrc string) {
			D.clientAddr[tunnelSrc] = transportSrc
		})
		if err != nil {
			serverDoneChan <- err
			break
		}
	}
}

func (D DatagramServer) tunLoop(server net.PacketConn) {
	buffer := make([]byte, 4096)
	for {
		n, err := D.Interface.Read(buffer)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		pkt := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if D.config.DropChatter && bizarre.IsChatter(pkt) {
			continue
		}
		fmt.Printf("\ntun=>net: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		netFlow := pkt.NetworkLayer().NetworkFlow()
		_, tunnelDst := netFlow.Endpoints()
		addr := D.clientAddr[tunnelDst.String()]
		if addr == nil {
			fmt.Print("No client addr found, skipping\n")
			continue
		}

		_, err = server.WriteTo(buffer[:n], addr)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		fmt.Printf("tun=>net: bytes=%d\n", n)
	}
}
