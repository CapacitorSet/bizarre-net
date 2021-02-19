package socket

import (
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
)

type transport struct {
	*net.UnixAddr
}

func NewTransport(config bizarre.Config, md toml.MetaData) (transport, error) {
	var srvConfig socketConfig
	err := md.PrimitiveDecode(config.Socket, &srvConfig)
	if err != nil {
		return transport{}, err
	}
	addr, err := net.ResolveUnixAddr("unix", srvConfig.Socket)
	if err != nil {
		return transport{}, err
	}
	return transport{addr}, nil
}

func (T transport) Listen() (net.Listener, error) {
	conn, err := net.ListenUnix("unix", T.UnixAddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (T transport) Dial() (net.Conn, error) {
	return net.DialUnix("unix", nil, T.UnixAddr)
}

func (T transport) HasIPRoutingConflict(bizarre.Interface) (bool, error) {
	return false, nil
}
