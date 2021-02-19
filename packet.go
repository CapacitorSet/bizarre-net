package bizarre_net

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

// Returns the packet (or nil if not an IP packet), and whether it is IPv6
func TryParse(packet []byte) gopacket.Packet {
	// Try parsing as IPv4, then as IPv6, then skip
	pkt := gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.Default)
	if pkt.ErrorLayer() == nil {
		return pkt
	}
	pkt = gopacket.NewPacket(packet, layers.LayerTypeIPv6, gopacket.Default)
	if pkt.ErrorLayer() == nil {
		return pkt
	}
	return nil
}

// Returns a string like "IPv4 TCP Payload"
func LayerString(pkt gopacket.Packet) string {
	if len(pkt.Layers()) == 0 {
		return ""
	}
	ret := ""
	for _, layer := range pkt.Layers() {
		ret += " " + layer.LayerType().String()
	}
	return ret[1:]
}

func FlowString(pkt gopacket.Packet) string {
	var flags, srcPort, dstPort, protoName string
	if tcpLayer := pkt.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		tcpFlags := ""
		if tcp.SYN {
			tcpFlags = tcpFlags + "S"
		}
		if tcp.ACK {
			tcpFlags = tcpFlags + "A"
		}
		if tcpFlags != "" {
			flags = " [" + tcpFlags + "]"
		}
		srcPort = tcp.SrcPort.String()
		dstPort = tcp.DstPort.String()
		protoName = "tcp"
	} else if udpLayer := pkt.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		srcPort = udp.SrcPort.String()
		dstPort = udp.DstPort.String()
		protoName = "udp"
	} else if pkt.Layer(layers.LayerTypeICMPv4) != nil || pkt.Layer(layers.LayerTypeICMPv6) != nil {
		protoName = "icmp"
	}
	var srcStr, dstStr string
	netFlow := pkt.NetworkLayer().NetworkFlow()
	src, dst := netFlow.Endpoints()
	if srcPort != "" {
		srcStr = net.JoinHostPort(src.String(), srcPort)
		dstStr = net.JoinHostPort(dst.String(), dstPort)
	} else {
		srcStr = src.String()
		dstStr = dst.String()
	}
	return fmt.Sprintf("%s/%s => %s/%s%s", srcStr, protoName, dstStr, protoName, flags)
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
