package ping

import (
	"errors"
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
	"os"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type transport struct {
	*net.IPAddr
}

func (T transport) IsReadable() bool { return true }

func (T transport) IsWritable() bool { return true }

type conn struct {
	*net.IPConn
}

func (c conn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	if err != nil {
		return 0, err
	}
	return
}

func (c conn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.RemoteAddr())
}

func (c conn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, peer, err := c.IPConn.ReadFrom(p)
	if err != nil {
		return 0, nil, err
	}
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), p[:n])
	if err != nil {
		return 0, nil, err
	}
	if rm.Type != ipv4.ICMPTypeEchoReply {
		return 0, nil, errors.New("unexpected ICMP type")
	}
	return n, peer, nil
}

func (c conn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// https://stackoverflow.com/a/27773040
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: p,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return 0, err
	}
	return c.IPConn.WriteTo(wb, addr)
}

var _ bizarre.ClientTransport = transport{}
var _ bizarre.PacketServer = transport{}

func (T transport) HasIPRoutingConflict(iface bizarre.Interface) (bool, error) {
	return iface.IsRoutedThrough(T.IP)
}

func (T transport) Listen() (net.PacketConn, error) {
	ipConn, err := net.ListenIP("icmp", T.IPAddr)
	if err != nil {
		return conn{}, err
	}
	return conn{ipConn}, nil
}

func (T transport) Dial() (net.Conn, error) {
	ipConn, err := net.DialIP("icmp", nil, T.IPAddr)
	if err != nil {
		return conn{}, err
	}
	return conn{ipConn}, err
}

func Client(config bizarre.Config, md toml.MetaData) (bizarre.ClientTransport, error) {
	var c pingConfig
	err := md.PrimitiveDecode(config.UDP, &c)
	if err != nil {
		return transport{}, err
	}
	addr, err := net.ResolveIPAddr("ip", c.IP)
	if err != nil {
		return transport{}, err
	}
	return transport{addr}, nil
}

func Server(config bizarre.Config, md toml.MetaData) (bizarre.PacketServer, error) {
	var c pingConfig
	err := md.PrimitiveDecode(config.UDP, &c)
	if err != nil {
		return transport{}, err
	}
	addr, err := net.ResolveIPAddr("ip", c.IP)
	if err != nil {
		return transport{}, err
	}
	return transport{addr}, nil
}
