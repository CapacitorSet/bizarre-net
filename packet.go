package bizarre_net

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Returns the packet (or nil if not an IP packet), and whether it is IPv6
func TryParse(packet []byte) (gopacket.Packet, bool) {
	// Try parsing as IPv4, then as IPv6, then skip
	pkt := gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.Default)
	if pkt.ErrorLayer() == nil {
		return pkt, false
	}
	pkt = gopacket.NewPacket(packet, layers.LayerTypeIPv6, gopacket.Default)
	if pkt.ErrorLayer() == nil {
		return pkt, true
	}
	return nil, false
}

func PrintPacket(pkt gopacket.Packet, isIPv6 bool) {
	fmt.Print("Packet layers:")
	for _, layer := range pkt.Layers() {
		fmt.Print(" " + layer.LayerType().String())
	}
	fmt.Println("")

	netFlow := pkt.NetworkLayer().NetworkFlow()
	src, dst := netFlow.Endpoints()
	var srcStr, dstStr string
	if isIPv6 {
		srcStr = "[" + src.String() + "]"
		dstStr = "[" + dst.String() + "]"
	} else {
		srcStr = src.String()
		dstStr = dst.String()
	}
	if tcpLayer := pkt.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		flags := ""
		if tcp.SYN {
			flags = flags + "S"
		}
		if tcp.ACK {
			flags = flags + "A"
		}
		if flags != "" {
			flags = "[" + flags + "]"
		}
		fmt.Printf("tcp %s: %s:%d to %s:%d\n", flags, srcStr, tcp.SrcPort, dstStr, tcp.DstPort)
	} else if udpLayer := pkt.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		fmt.Printf("udp: %s:%d to %s:%d\n", srcStr, udp.SrcPort, dstStr, udp.DstPort)
	}
}

func IsChatter(packet gopacket.Packet) bool {
	// Todo: match broadcast addresses instead
	// LLMNR
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		if udpLayer.(*layers.UDP).SrcPort == 5355 {
			return true
		}
	} else if packet.Layer(layers.LayerTypeIGMP) != nil {
		return true
	} else if packet.Layer(layers.LayerTypeICMPv6NeighborSolicitation) != nil {
		return true
	} else if packet.Layer(layers.LayerTypeICMPv6RouterSolicitation) != nil {
		return true
	} else if packet.Layer(layers.LayerTypeMLDv2MulticastListenerReport) != nil {
		return true
	}
	return false
}
