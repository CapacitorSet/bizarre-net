package bizarre_net

import (
	"net"
)

type Transport interface {
	Dial() (net.Conn, error)
	HasIPRoutingConflict(Interface) (bool, error)
}

// UDP-like transports: connectionless, stateless
type DatagramTransport interface {
	Transport
	Listen() (net.PacketConn, error)
}

// TCP-like transports
type StreamTransport interface {
	Transport
	Listen() (net.Listener, error)
}
