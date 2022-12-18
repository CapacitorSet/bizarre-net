package transports

import (
	"io"
	"net"
)

type UDPConfig struct {
	Endpoint string // The UDP address to connect to
}

type UDPServerTransport struct {
	Conn net.UDPConn
}

func (T *UDPServerTransport) Listen(ch chan<- Packet) {
	buffer := make([]byte, 1500)
	for {
		n, addr, err := T.Conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}
		ch <- Packet{Payload: buffer[:n], Address: addr}
	}
}

func (T *UDPServerTransport) WriteTo(payload []byte, address interface{}) (int, error) {
	return T.Conn.WriteToUDP(payload, address.(*net.UDPAddr))
}

type UDPWriter struct {
	*UDPServerTransport
	Destination *net.UDPAddr
}

func (w UDPWriter) Write(p []byte) (int, error) {
	return w.UDPServerTransport.WriteTo(p, w.Destination)
}

// WriterTo returns an io.Writer that writes to an address
func (T *UDPServerTransport) WriterTo(address interface{}) io.Writer {
	return UDPWriter{T, address.(*net.UDPAddr)}
}

type UDPClientTransport struct {
	Conn net.UDPConn
}

func (T *UDPClientTransport) Listen(ch chan<- []byte) {
	buffer := make([]byte, 1500)
	for {
		n, err := T.Conn.Read(buffer)
		if err != nil {
			panic(err)
		}
		ch <- buffer[:n]
	}
}

func (T *UDPClientTransport) Write(payload []byte) (int, error) {
	return T.Conn.Write(payload)
}

func CreateUDPServer(config UDPConfig) (UDPServerTransport, error) {
	if config.Endpoint == "" {
		return UDPServerTransport{}, nil
	}

	addr, err := net.ResolveUDPAddr("udp", config.Endpoint)
	if err != nil {
		return UDPServerTransport{}, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return UDPServerTransport{}, err
	}

	return UDPServerTransport{Conn: *conn}, nil
}

func CreateUDPClient(config UDPConfig) (UDPClientTransport, error) {
	if config.Endpoint == "" {
		return UDPClientTransport{}, nil
	}

	addr, err := net.ResolveUDPAddr("udp", config.Endpoint)
	if err != nil {
		return UDPClientTransport{}, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return UDPClientTransport{}, err
	}

	return UDPClientTransport{Conn: *conn}, nil
}