package socket

import (
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
)

type Transport struct{}

func (T Transport) Listen(config bizarre.Config, md toml.MetaData) (net.Listener, error) {
	var srvConfig socketConfig
	err := md.PrimitiveDecode(config.Socket, &srvConfig)
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUnixAddr("unix", srvConfig.Socket)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (T Transport) Dial(config bizarre.Config, md toml.MetaData) (net.Conn, error) {
	var cltConfig socketConfig
	err := md.PrimitiveDecode(config.Socket, &cltConfig)
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUnixAddr("unix", cltConfig.Socket)
	if err != nil {
		return nil, err
	}
	return net.DialUnix("unix", nil, addr)
}
