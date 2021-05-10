package cat

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	bizarre "github.com/CapacitorSet/bizarre-net"
	"net"
	"os"
	"time"
)

type transport struct {
	serverConfig
	clientConfig
}

func NewServerTransport(config bizarre.Config, md toml.MetaData) (transport, error) {
	var srvConfig serverConfig
	err := md.PrimitiveDecode(config.Cat, &srvConfig)
	if err != nil {
		return transport{}, err
	}
	if srvConfig.ServerName == "" {
		return transport{}, errors.New("empty server name")
	}
	return transport{serverConfig: srvConfig}, nil
}

func NewClientTransport(config bizarre.Config, md toml.MetaData) (transport, error) {
	var cltConfig clientConfig
	err := md.PrimitiveDecode(config.UDP, &cltConfig)
	if err != nil {
		return transport{}, err
	}
	return transport{clientConfig: cltConfig}, nil
}

type Addr string

func (a Addr) Network() string { return "cat" }
func (a Addr) String() string  { return string(a) }

type Connection struct {
	localAddr, remoteAddr Addr

	stdinReader *bufio.Reader
}

func (c Connection) Read(b []byte) (n int, err error) {
	fmt.Print("Packet contents: ")
	contents, err := c.stdinReader.ReadString('\n')
	if err != nil {
		return 0, err
	}

	return copy(b, contents), nil
}

func (c Connection) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.remoteAddr)
}

func (c Connection) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c Connection) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	fmt.Print("Packet source: ")
	from, err := c.stdinReader.ReadString('\n')
	if err != nil {
		return 0, nil, err
	}
	n, err = c.Read(p)
	if err != nil {
		return 0, nil, err
	}
	return n, Addr(from), nil
}

func (c Connection) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	fmt.Printf("Please fetch the cat and write the following message to %s: %#v.\n", addr, p)
	fmt.Println("Press any key to continue when you're done.")
	c.stdinReader.ReadLine()
	return len(p), nil
}

func (c Connection) Close() error {
	panic("not implemented")
}

func (c Connection) LocalAddr() net.Addr {
	return c.localAddr
}

func (c Connection) SetDeadline(t time.Time) error {
	panic("not implemented")
}

func (c Connection) SetReadDeadline(t time.Time) error {
	panic("not implemented")
}

func (c Connection) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
}

func (T transport) Listen() (net.PacketConn, error) {
	return Connection{
		stdinReader: bufio.NewReader(os.Stdin),
		localAddr:   Addr(T.serverConfig.ServerName),
	}, nil
}

func (T transport) Dial() (net.Conn, error) {
	return Connection{
		stdinReader: bufio.NewReader(os.Stdin),
		localAddr:   Addr(T.clientConfig.ClientName),
		remoteAddr:  Addr(T.clientConfig.ServerName),
	}, nil
}

func (T transport) HasIPRoutingConflict(bizarre.Interface) (bool, error) {
	return false, nil
}
