package bizarre_net

import (
	"net"
)

type Transport interface {
	Dial() (net.Conn, error)
	HasIPRoutingConflict(Interface) (bool, error)
}

// DatagramTransport is a UDP-like transport: connectionless, stateless
type DatagramTransport interface {
	Transport
	Listen() (net.PacketConn, error)
}

// StreamTransport is a TCP-like transport
type StreamTransport interface {
	Transport
	Listen() (net.Listener, error)
}
