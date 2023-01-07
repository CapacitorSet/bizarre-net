package transports

import (
	"flag"
	"fmt"
	"io"
	"log"
)

type Packet struct {
	Payload []byte
	Address interface{}
}

type ServerTransport interface {
	Listen(ch chan<- Packet)
	WriteTo(payload []byte, address interface{}) (int, error)
	WriterTo(address interface{}) io.Writer // todo: deduplicate this
}

type ClientTransport interface {
	Listen(ch chan<- []byte)
	Write(payload []byte) (int, error)
}

type TransportConfig struct {
	UDPConfig UDPConfig
	DNSConfig DNSConfig
}

// PartialConfigFromFlags binds a flagset to a TransportConfig struct, so that the config is filled upon parsing the flags.
func PartialConfigFromFlags(config *TransportConfig, flags *flag.FlagSet) {
	flags.StringVar(&config.DNSConfig.Endpoint, "dns-address", "", "DNS server address")
	flags.IntVar(&config.DNSConfig.Port, "dns-port", 53, "DNS server port")
	flags.StringVar(&config.DNSConfig.RootDomain, "dns-root", ".biz", "DNS root domain including TLD")
	flags.StringVar(&config.UDPConfig.Endpoint, "udp-address", "", "UDP server address")
}

// NewServerTransport creates a ServerTransport from a TransportConfig.
func NewServerTransport(config TransportConfig) (ServerTransport, error) {
	if config.UDPConfig.Endpoint != "" {
		udp, err := CreateUDPServer(config.UDPConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("Listening on UDP with IP %s\n", config.UDPConfig.Endpoint)
		return &udp, nil
	} else if config.DNSConfig.Port != 0 {
		dns, err := CreateDNSServer(config.DNSConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("Listening on DNS with port %d\n", config.DNSConfig.Port)
		return &dns, nil
	} else {
		return nil, fmt.Errorf("no transport selected")
	}
}

// NewClientTransport creates a ClientTransport from a TransportConfig.
func NewClientTransport(config TransportConfig) (ClientTransport, error) {
	if config.UDPConfig.Endpoint != "" {
		udp, err := CreateUDPClient(config.UDPConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("Using UDP transport with IP %s\n", config.UDPConfig.Endpoint)
		return &udp, nil
	} else if config.DNSConfig.Endpoint != "" {
		udp, err := CreateDNSClient(config.DNSConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("Using DNS transport with IP %s\n", config.DNSConfig.Endpoint)
		return &udp, nil
	} else {
		return nil, fmt.Errorf("no transport selected")
	}
}
