package bizarre_net

import (
	"net"
)

type baseTransport interface {
	HasIPRoutingConflict(Interface) (bool, error)
	IsReadable() bool
	IsWritable() bool
}

type ClientTransport interface {
	baseTransport
	Dial() (net.Conn, error)
}

type ConnServer interface {
	baseTransport
	Listen() (net.Listener, error)
}

type PacketServer interface {
	baseTransport
	Listen() (net.PacketConn, error)
}

// ServerTransport is one of ConnServer or PacketServer.
type ServerTransport struct {
	ConnServer
	PacketServer
}