package udp

import (
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
)

type Transport struct{}

func (T Transport) Listen(config bizarre.Config, md toml.MetaData) (net.PacketConn, error) {
	var udpSrvConfig udpConfig
	err := md.PrimitiveDecode(config.UDP, &udpSrvConfig)
	if err != nil {
		return nil, err
	}
	serverAddr, err := net.ResolveUDPAddr("udp", udpSrvConfig.IP)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", serverAddr)
}

func (T Transport) Dial(config bizarre.Config, md toml.MetaData) (net.Conn, error) {
	var udpClientConfig udpConfig
	err := md.PrimitiveDecode(config.UDP, &udpClientConfig)
	if err != nil {
		return nil, err
	}
	serverAddr, err := net.ResolveUDPAddr("udp", udpClientConfig.IP)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, serverAddr)
}
