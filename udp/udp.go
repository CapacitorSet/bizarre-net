package udp

import (
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
)

type transport struct {
	*net.UDPAddr
}

func (T transport) HasIPRoutingConflict(iface bizarre.Interface) (bool, error) {
	return iface.IsRoutedThrough(T.IP)
}

func NewTransport(config bizarre.Config, md toml.MetaData) (transport, error) {
	var udpSrvConfig udpConfig
	err := md.PrimitiveDecode(config.UDP, &udpSrvConfig)
	if err != nil {
		return transport{}, err
	}
	addr, err := net.ResolveUDPAddr("udp", udpSrvConfig.IP)
	if err != nil {
		return transport{}, err
	}
	return transport{addr}, nil
}

func (T transport) Listen() (net.PacketConn, error) {
	return net.ListenUDP("udp", T.UDPAddr)
}

func (T transport) Dial() (net.Conn, error) {
	return net.DialUDP("udp", nil, T.UDPAddr)
}
