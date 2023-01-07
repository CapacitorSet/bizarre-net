package transports

import (
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

var (
	_ ServerTransport = (*DNSServerTransport)(nil)
	_ ClientTransport = (*DNSClientTransport)(nil)

	Encoder = base32.HexEncoding.WithPadding(base32.NoPadding)
)

type DNSConfig struct {
	Endpoint string // The DNS server to connect to
	Port int

	RootDomain string // The DNS domain to be appended
}

type DNSServerTransport struct {
	Server dns.Server
	RootDomain string

	ch chan<- Packet
}

func (T *DNSServerTransport) handleDnsRequest(rw dns.ResponseWriter, msg *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(msg)
	m.Compress = false
	domain := strings.TrimSuffix(msg.Question[0].Name, T.RootDomain)
	data, err := Encoder.DecodeString(domain)
	if err != nil {
		log.Printf("could not interpret string: %s", err)
		rw.WriteMsg(m)
		return
	}

	T.ch <- Packet{
		Payload: data,
		Address: nil,
	}

	// Skip parsing: assume that the message is a TXT query
	text := "TODOwriteme"
	rr, err := dns.NewRR(fmt.Sprintf("%s TXT %s", "bizarre-net.capacitorset.github.com", text))
	if err == nil {
		m.Answer = append(m.Answer, rr)
	}

	rw.WriteMsg(m)
}

func (T *DNSServerTransport) Listen(ch chan<- Packet) {
	T.ch = ch
	dns.HandleFunc(".", T.handleDnsRequest)
	T.Server.ListenAndServe()
}

func (T *DNSServerTransport) WriteTo(payload []byte, address interface{}) (int, error) {
	return 0, nil
	// return T.Conn.WriteToDNS(payload, address.(*net.DNSAddr))
}

type DNSWriter struct {
	*DNSServerTransport
}

func (w DNSWriter) Write(p []byte) (int, error) {
	log.Printf("Notice: DNSWriter is not implemented yet")
	return 0, nil
	// return w.DNSServerTransport.WriteTo(p, w.Destination)
}

// WriterTo returns an io.Writer that writes to an address
func (T *DNSServerTransport) WriterTo(address interface{}) io.Writer {
	return DNSWriter{T}
	// return DNSWriter{T, address.(*net.Addr)}
}

type DNSClientTransport struct {
	Endpoint string
	RootDomain string
}

func (T *DNSClientTransport) Listen(ch chan<- []byte) {
	/*
	buffer := make([]byte, 1500)
	for {
		n, err := T.Conn.Read(buffer)
		if err != nil {
			panic(err)
		}
		ch <- buffer[:n]
	}
	*/
}

func (T *DNSClientTransport) Write(payload []byte) (int, error) {
	// Encode the payload as base32 (which is DNS-safe) and add a root domain for correct routing (can be just "." if there are no relays)
	domain := Encoder.EncodeToString(payload) + T.RootDomain
	if len(payload) > 255 {
		return 0, fmt.Errorf("payload too long for DNS")
	}
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeTXT)
	reply, err := dns.Exchange(m, T.Endpoint)
	_ = reply
	return len(payload), err
}

func CreateDNSServer(config DNSConfig) (DNSServerTransport, error) {
	return DNSServerTransport{
		Server: dns.Server{
			Addr: ":" + strconv.Itoa(int(config.Port)),
			Net: "udp",
		},
		RootDomain: config.RootDomain + ".",
	}, nil
}

func CreateDNSClient(config DNSConfig) (DNSClientTransport, error) {
	l := 0
	mtu := 0
	// Find the MTU such that the whole domain is under 255 bytes
	for l < 255 {
		mtu += 1
		l = Encoder.EncodedLen(mtu) + len(config.RootDomain + ".")
	}
	log.Printf("MTU for DNS should be %d", mtu)
	// Minimum MTU is 68 per RFC (60 bytes of IP header + 8 bytes of payload)
	if mtu < 68 {
		return DNSClientTransport{}, fmt.Errorf("MTU too low: %d < 68", mtu)
	}
	return DNSClientTransport{
		Endpoint: fmt.Sprintf("%s:%d", config.Endpoint, config.Port),
		RootDomain: config.RootDomain + ".",
	}, nil
}