package transports

import (
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

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

	SendQueue

	ch chan<- Packet
}

// SendQueue is a structure that stores packets waiting to be sent.
// We use it for request-response protocols like DNS where we can only send
// packets at specific instants
type SendQueue struct {
	sync.Mutex
	queue []Packet
}

func (Q *SendQueue) Push(packet Packet) {
	Q.Lock()
	Q.queue = append(Q.queue, packet)
	log.Printf("New queue length: %d", len(Q.queue))
	Q.Unlock()
}

func (Q *SendQueue) TryGet() (ok bool, p Packet) {
	Q.Lock()
	if len(Q.queue) == 0 {
		Q.Unlock()
		return false, Packet{}
	}
	ret := Q.queue[0]
	Q.queue = Q.queue[1:]
	log.Printf("New queue length: %d", len(Q.queue))
	Q.Unlock()
	return true, ret
}

func (T *DNSServerTransport) handleDnsRequest(rw dns.ResponseWriter, msg *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(msg)
	m.Compress = false
	domain := strings.TrimSuffix(msg.Question[0].Name, "." + T.RootDomain)
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

	// Try to find something in the send queue
	ok, pkt := T.SendQueue.TryGet()
	i := 0
	// If the queue is empty, keep trying over the next second
	for !ok && i < 10 {
		i += 1
		time.Sleep(100 * time.Millisecond)
		ok, pkt = T.SendQueue.TryGet()
	}

	if ok {
		// Skip parsing: assume that the message is a TXT query
		text := Encoder.EncodeToString(pkt.Payload)
		rr, err := dns.NewRR(fmt.Sprintf("%s TXT %s", "bizarre-net.capacitorset.github.com", text))
		if err != nil {
			log.Printf("Failed to send data: %s", err)
		} else {
			m.Answer = append(m.Answer, rr)
		}
	}

	rw.WriteMsg(m)
}

func (T *DNSServerTransport) Listen(ch chan<- Packet) {
	T.ch = ch
	dns.HandleFunc(".", T.handleDnsRequest)
	T.Server.ListenAndServe()
}

func (T *DNSServerTransport) WriteTo(payload []byte, address interface{}) (int, error) {
	T.SendQueue.Push(Packet{
		Payload: payload,
		Address: address,
	})
	// todo: wait for the packet to be sent
	return len(payload), nil
}

type DNSWriter struct {
	*DNSServerTransport
	address interface{}
}

func (w DNSWriter) Write(p []byte) (int, error) {
	return w.DNSServerTransport.WriteTo(p, w.address)
}

// WriterTo returns an io.Writer that writes to an address
func (T *DNSServerTransport) WriterTo(address interface{}) io.Writer {
	return DNSWriter{T, address}
	// return DNSWriter{T, address.(*net.Addr)}
}

type DNSClientTransport struct {
	Endpoint string
	RootDomain string

	ch chan<- []byte
}

func (T *DNSClientTransport) Listen(ch chan<- []byte) {
	T.ch = ch
}

func (T *DNSClientTransport) Write(payload []byte) (int, error) {
	// Encode the payload as base32 (which is DNS-safe) and add a root domain for correct routing (can be just "." if there are no relays)
	domain := Encoder.EncodeToString(payload) + "." + T.RootDomain
	if len(payload) > 255 {
		return 0, fmt.Errorf("payload too long for DNS")
	}
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeTXT)
	reply, err := dns.Exchange(m, T.Endpoint)
	if len(reply.Answer) != 0 {
		if t, ok := reply.Answer[0].(*dns.TXT); ok {
			data, err := Encoder.DecodeString(t.Txt[0])
			if err != nil {
				log.Printf("Failed to parse reply: %s", err)
			} else {
				log.Printf("Sending %x on %#v", data, T.ch)
				T.ch <- data
			}
		} else {
			log.Printf("Response is not a TXT record")
		}
	}
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