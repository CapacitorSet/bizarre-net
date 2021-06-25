package server

import (
	"bytes"
	"fmt"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"log"
	"net"
)

type ConnServer struct {
	BaseServer
	Transport bizarre.ConnServer

	// Maps the in-tunnel source IP of the host to its connection (used in Write for stream transports)
	clientConn map[string]net.Conn
}

func (C ConnServer) Run() error {
	serverDoneChan := make(chan error)
	server, err := C.Transport.Listen()
	if err != nil {
		return err
	}
	defer server.Close()
	go C.serverLoop(server, serverDoneChan)
	go C.tunStreamLoop()

	select {
	case err := <-serverDoneChan:
		return err
	}
}

// Accepts connections for stream transports
func (C ConnServer) serverLoop(listener net.Listener, serverDoneChan chan error) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}
		go C.connLoop(conn, serverDoneChan)
	}
}

// Handles packets from a datagram transport or from a TCP-like connection
func (C ConnServer) connLoop(conn net.Conn, serverDoneChan chan error) {
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

		// Neither an IPv4 nor an IPv6 packet
		if buffer[0]&0xf0 != 0x40 && buffer[0]&0xf0 != 0x60 {
			fmt.Printf("\nnet=>tun: service message, bytes=%d\n", n)
			if bytes.Equal(buffer[:n], bizarre.HELLO_MESSAGE) {
				_, err = conn.Write(bizarre.HELLO_ACK_MESSAGE)
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

		err = C.processNetPkt(buffer[:n], func(tunnelSrc string) {
			C.clientConn[tunnelSrc] = conn
		})
		if err != nil {
			log.Println(err)
			serverDoneChan <- err
			break
		}
	}
}

func (C ConnServer) tunStreamLoop() {
	buffer := make([]byte, 4096)
	for {
		n, err := C.Interface.Read(buffer)
		if err != nil {
			log.Printf("tunLoop: " + err.Error())
			continue
		}

		pkt := bizarre.TryParse(buffer[:n])
		if pkt == nil {
			log.Println("Skipping packet, can't parse as IPv4 nor IPv6")
			continue
		}
		if C.Config.DropChatter && bizarre.IsChatter(pkt) {
			continue
		}
		fmt.Printf("\ntun=>net: %s %s bytes=%d\n", bizarre.FlowString(pkt), bizarre.LayerString(pkt), n)

		netFlow := pkt.NetworkLayer().NetworkFlow()
		_, tunnelDst := netFlow.Endpoints()
		conn := C.clientConn[tunnelDst.String()]
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
