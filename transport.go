package bizarre_net

import (
	"github.com/BurntSushi/toml"
	"net"
)

type Transport interface {
	Dial(config Config, md toml.MetaData) (net.Conn, error)
}

// UDP-like transports: connectionless, stateless
type DatagramTransport interface {
	Transport
	Listen(config Config, md toml.MetaData) (net.PacketConn, error)
}

// TCP-like transports
type StreamTransport interface {
	Transport
	Listen(config Config, md toml.MetaData) (net.Listener, error)
}
