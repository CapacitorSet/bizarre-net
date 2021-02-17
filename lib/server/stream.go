package server

import (
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

// Accepts connections for stream transports
func (S Server) streamLoop(listener net.Listener, serverDoneChan chan error) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}
		go S.streamConnLoop(conn, serverDoneChan)
	}
}

// Handles packets from a datagram transport or from a TCP-like connection
func (S Server) streamConnLoop(conn net.Conn, serverDoneChan chan error) {
	buffer := make([]byte, 1500)
	for {
		// By reading from the connection into the buffer, we block until there's
		// new content in the socket that we're listening for new packets.

		// Note: `buffer` is not being reset between runs, so you must read only `n` bytes.
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}

		err = S.processNetPkt(buffer[:n], conn.RemoteAddr(), func(tunnelSrc string) {
			S.clientConn[tunnelSrc] = conn
		})
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}
	}
}

func (S Server) tunStreamLoop(iface bizarre.Interface) {
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
		conn := S.clientConn[tunnelDst.String()]
		if conn == nil {
			fmt.Print("No client conn found, skipping\n")
			continue
		}

		_, err = conn.Write(buffer[:n])
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		fmt.Printf("> net bytes=%d\n", n)
	}
}
