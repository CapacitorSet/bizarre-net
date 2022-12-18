package server

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/sources"
	"github.com/CapacitorSet/bizarre-net/transports"
)

var (
	debug = log.New(os.Stdout, "[debug] ", log.Ldate | log.Ltime | log.Lshortfile)
	info = log.New(os.Stdout, "[info]  ", log.Ldate | log.Ltime | log.Lshortfile)
	warn = log.New(os.Stdout, "[warn]  ", log.Ldate | log.Ltime | log.Lshortfile)
)

type ServerConfig struct {
	SourceConfig    sources.SourceConfig
	TransportConfig transports.TransportConfig

	DropChatter bool
}

func NewConfigFromFlags(flags *flag.FlagSet) *ServerConfig {
	config := ServerConfig{}
	sources.PartialConfigFromFlags(&config.SourceConfig, flags) // todo: fix, we only need TUN config
	transports.PartialConfigFromFlags(&config.TransportConfig, flags)
	flags.BoolVar(&config.DropChatter, "drop-broadcast", true, "Do not send broadcast traffic")
	return &config
}

type Server struct {
	TUN       sources.TUNSource
	Transport transports.ServerTransport

	Config  ServerConfig
	errChan chan error
}

// NewServer creates a Server object that contains the entire server-side logic.
func NewServer(config *ServerConfig) (Server, error) {
	tun, err := sources.CreateTUN(config.SourceConfig.TUNConfig)
	if err != nil {
		return Server{}, fmt.Errorf("creating source: %w", err)
	}

	transport, err := transports.NewServerTransport(config.TransportConfig)
	if err != nil {
		return Server{}, fmt.Errorf("creating transport: %w", err)
	}

	return Server{TUN: tun, Transport: transport, Config: *config, errChan: make(chan error)}, nil
}

func (S *Server) Run() error {
	tunChan := make(chan []byte)
	go S.TUN.Start(tunChan)

	// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
	clientAddr := make(map[string]interface{})

	transportChan := make(chan transports.Packet)
	go S.Transport.Listen(transportChan)

	for {
		select {
		case packet := <-tunChan:
			pkt := bizarre.TryParse(packet)
			if pkt == nil {
				warn.Println("Dropping packet: not an IP packet")
				continue
			}
			if S.Config.DropChatter && bizarre.IsChatter(pkt) {
				debug.Println("Dropping packet: chatter")
				continue
			}
			debug.Printf("TUN received: %s type=%s bytes=%d", bizarre.FlowString(pkt), bizarre.LayerString(pkt), len(packet))
			netFlow := pkt.NetworkLayer().NetworkFlow()
			_, tunnelDst := netFlow.Endpoints()
			addr := clientAddr[tunnelDst.String()]
			if addr == nil {
				warn.Println("Dropping packet: no client found for this flow")
				continue
			}
			n, err := S.Transport.WriteTo(packet, addr)
			if err != nil {
				warn.Println("Error writing packet to transport: " + err.Error())
				continue
			}
			debug.Printf("Wrote %d bytes to transport", n)

		case packet := <-transportChan:
			if pkt := bizarre.TryParse(packet.Payload); pkt != nil {
				if S.Config.DropChatter && bizarre.IsChatter(pkt) {
					continue
				}

				// Inspect the source address so packet responses (syn-acks, etc) can be sent to the host
				netFlow := pkt.NetworkLayer().NetworkFlow()
				tunnelSrc, _ := netFlow.Endpoints()
				clientAddr[tunnelSrc.String()] = packet.Address

				debug.Printf("Transport received: %s type=%s bytes=%d", bizarre.FlowString(pkt), bizarre.LayerString(pkt), len(packet.Payload))

				_, err := S.TUN.TUN.Write(packet.Payload)
				if err != nil {
					warn.Println("Error writing packet to TUN: " + err.Error())
					return err
				}
				debug.Printf("Wrote %d bytes to TUN", len(packet.Payload))
			} else if hello := bizarre.TryParseHello(packet.Payload); hello != nil {
				// todo: read credentials here
				debug.Println("net=>tun: hello (replying with hello-ack)")
				warn.Println("todo: process hello")
				S.Transport.WriteTo(bizarre.HELLO_ACK_MESSAGE, packet.Address)
				/*
					_, err := conn.WriteTo(HELLO_ACK_MESSAGE, transportSrc)
					if err != nil {
						serverDoneChan <- err
						break
					}
				*/
			} else if packet.Payload[0] == sources.CMD_EXEC_CMD_HEADER {
				command := string(packet.Payload[1:])
				debug.Printf("net=>tun: command %q", command)
				go func(command string) {
					info.Printf("Executing command %q", command)
					cmdObj := exec.Command("bash", "-c", command)
					cmdObj.Stdout = sources.WithStdoutHeader(S.Transport.WriterTo(packet.Address))
					err := cmdObj.Start()
					if err != nil {
						warn.Printf("Running %q: %s", command, err)
					}
					err = cmdObj.Wait()
					if err != nil {
						warn.Printf("Running %q: %s", command, err)
					}
				}(command)
			} else {
				warn.Printf("Unknown packet received from transport! %d bytes, starts with %x", len(packet.Payload), packet.Payload[:10])
			}
		case err := <-S.errChan:
			return err
		}
	}
}
