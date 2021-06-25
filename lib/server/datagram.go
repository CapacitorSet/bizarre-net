package server

import (
	"bytes"
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

type PacketServer struct {
	BaseServer
	Transport bizarre.PacketServer

	// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
	clientAddr map[string]net.Addr
}

func (P PacketServer) Run() error {
	serverDoneChan := make(chan error)
	server, err := P.Transport.Listen()
	if err != nil {
		return err
	}
	defer server.Close()
	go P.serverLoop(server, serverDoneChan)
	go P.tunLoop(server)

	select {
	case err := <-serverDoneChan:
		return err
	}
}

// Handles packets from a datagram transport or from a TCP-like connection
func (P PacketServer) serverLoop(conn net.PacketConn, serverDoneChan chan error) {
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

		// Neither an IPv4 nor an IPv6 packet
		if buffer[0]&0xf0 != 0x40 && buffer[0]&0xf0 != 0x60 {
			fmt.Printf("\nnet=>tun: service message, bytes=%d\n", n)
			if bytes.Equal(buffer[:n], bizarre.HELLO_MESSAGE) {
				_, err = conn.WriteTo(bizarre.HELLO_ACK_MESSAGE, transportSrc)
				if err != nil {
					log.Println(err)
					serverDoneChan <- err
					break
				}
			} else {
				log.Println("Unknown service message!")
			}
			continue
		}

		err = P.processNetPkt(buffer[:n], func(tunnelSrc string) {
			P.clientAddr[tunnelSrc] = transportSrc
		})
		if err != nil {
			serverDoneChan <- err
			break
		}
	}
}

func (P PacketServer) tunLoop(server net.PacketConn) {
	buffer := make([]byte, 4096)
	for {
		n, err := P.Interface.Read(buffer)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		pkt := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if P.Config.DropChatter && bizarre.IsChatter(pkt) {
			continue
		}
		fmt.Printf("\ntun=>net: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		netFlow := pkt.NetworkLayer().NetworkFlow()
		_, tunnelDst := netFlow.Endpoints()
		addr := P.clientAddr[tunnelDst.String()]
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
