package bizarre_net

import (
	"github.com/BurntSushi/toml"
	"net"
)

type Transport interface {
	Listen(config Config, md toml.MetaData) (net.PacketConn, error)
	Dial(config Config, md toml.MetaData) (net.Conn, error)
}