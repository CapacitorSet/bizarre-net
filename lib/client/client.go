package client

import (
	"flag"
	"fmt"
	"log"
	"os"

	bizarre "github.com/CapacitorSet/bizarre-net"
	"github.com/CapacitorSet/bizarre-net/sources"
	"github.com/CapacitorSet/bizarre-net/transports"
)

var (
	debug = log.New(os.Stdout, "[debug] ", log.Ldate | log.Ltime | log.Lshortfile)
	info = log.New(os.Stdout, "[info]  ", log.Ldate | log.Ltime | log.Lshortfile)
	warn = log.New(os.Stdout, "[warn]  ", log.Ldate | log.Ltime | log.Lshortfile)
)

type ClientConfig struct {
	SourceConfig    sources.SourceConfig
	TransportConfig transports.TransportConfig

	DropChatter bool
	SendHello bool
}

func NewConfigFromFlags(flags *flag.FlagSet) *ClientConfig {
	config := ClientConfig{}
	sources.PartialConfigFromFlags(&config.SourceConfig, flags) // todo: fix, we only need TUN config
	transports.PartialConfigFromFlags(&config.TransportConfig, flags)
	flags.BoolVar(&config.DropChatter, "drop-broadcast", true, "Do not send broadcast traffic")
	// todo: figure out how to encode flag
	config.SendHello = true
	return &config
}

type Client struct {
	Source sources.Source
	Transport transports.ClientTransport

	Config ClientConfig
	errChan  chan error
}

// NewClient creates a Server object that contains the entire client-side logic.
func NewClient(config *ClientConfig) (Client, error) {
	source, err := sources.NewSource(config.SourceConfig)
	if err != nil {
		return Client{}, fmt.Errorf("creating source: %w", err)
	}

	// Todo: check if endpoint IP is routed via TUN
	transport, err := transports.NewClientTransport(config.TransportConfig)
	if err != nil {
		return Client{}, fmt.Errorf("creating transport: %w", err)
	}

	return Client{Source: source, Transport: transport, Config: *config, errChan: make(chan error)}, nil
}

func (C Client) Run() error {
	sourceChan := make(chan []byte)
	go C.Source.Start(sourceChan)

	// Maps the in-tunnel source IP of the host to its transport address (used in WriteTo for datagram transports)
	// clientAddr := make(map[string]interface{})

	transportChan := make(chan []byte)
	go C.Transport.Listen(transportChan)

	if C.Config.SendHello {
		debug.Println("Sending hello")
		// todo: WriteToServer
		_, err := C.Transport.Write(bizarre.HELLO_PREFIX)
		if err != nil {
			warn.Printf("Could not send hello: %s", err)
			return err
		}
	}

	for {
		select {
		case packet := <-sourceChan:
			if pkt := bizarre.TryParse(packet); pkt != nil {
				if C.Config.DropChatter && bizarre.IsChatter(pkt) {
					debug.Println("Dropping packet: chatter")
					continue
				}
				debug.Printf("Source received: %s type=%s bytes=%d", bizarre.FlowString(pkt), bizarre.LayerString(pkt), len(packet))
			} else if packet[0] == sources.CMD_EXEC_CMD_HEADER {
				debug.Printf("Source received: CmdExec packet bytes=%d", len(packet)-1)
			} else {
				print_len := 10
				if len(packet) < print_len {
					print_len = len(packet)
				}
				warn.Printf("Dropping packet: not an IP packet (begins with %x)", packet[:print_len])
				continue
			}
			// todo: WriteToServer
			n, err := C.Transport.Write(packet)
			if err != nil {
				warn.Println("Error writing packet to transport: " + err.Error())
				continue
			}
			debug.Printf("Wrote %d bytes to transport", n)
		case packet := <-transportChan:
			if pkt := bizarre.TryParse(packet); pkt != nil {
				if C.Config.DropChatter && bizarre.IsChatter(pkt) {
					continue
				}

				debug.Printf("Transport received: %s type=%s bytes=%d", bizarre.FlowString(pkt), bizarre.LayerString(pkt), len(packet))

				err := C.Source.Write(packet)
				if err != nil {
					warn.Println("Error writing packet to TUN: " + err.Error())
					return err
				}
				debug.Printf("Wrote %d bytes to TUN", len(packet))
			} else if hello := bizarre.TryParseHelloAck(packet); hello {
				debug.Println("net=>tun: hello-ack")
				warn.Println("todo: process hello-ack")
			} else if packet[0] == sources.CMD_EXEC_STDOUT_HEADER {
				info.Printf("Command output: %s", packet[1:])
			} else {
				warn.Printf("Unknown packet received from transport! %d bytes, starts with %x", len(packet), packet[:10])
			}
		case err := <-C.errChan:
			return err
		}
	}
}
