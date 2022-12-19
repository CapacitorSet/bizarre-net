package bizarre_net

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// TryParse returns a parsed IP packet
func TryParse(packet []byte) gopacket.Packet {
	// Fast check; it also "fixes" IPv6 detection (Gopacket parses a packet starting with CE 36 2E 30 38 33 3A 39 as IPv6)
	if (packet[0] & 0xF0) != 0x60 && (packet[0] & 0xF0) != 0x40 {
		return nil
	}
	// Try parsing as IPv4, then as IPv6, then skip
	if pkt := gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.Default); pkt.ErrorLayer() == nil {
		return pkt
	}
	if pkt := gopacket.NewPacket(packet, layers.LayerTypeIPv6, gopacket.Default); pkt.ErrorLayer() == nil {
		return pkt
	}
	return nil
}

// LayerString returns the specific type of packet (eg. "TCP", "ICMPv6RouterSolicitation")
func LayerString(pkt gopacket.Packet) string {
	layers := pkt.Layers()
	if len(layers) == 0 {
		return ""
	}
	finalLayer := layers[len(layers)-1]
	if finalLayer.LayerType() == gopacket.LayerTypePayload {
		finalLayer = layers[len(layers)-2]
	}
	return finalLayer.LayerType().String()
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
			flags = " flags=" + tcpFlags
		}
		srcPort = fmt.Sprint(uint16(tcp.SrcPort))
		dstPort = fmt.Sprint(uint16(tcp.DstPort))
		protoName = "tcp"
	} else if udpLayer := pkt.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		srcPort = fmt.Sprint(uint16(udp.SrcPort))
		dstPort = fmt.Sprint(uint16(udp.DstPort))
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
	return fmt.Sprintf("%s => %s proto=%s%s", srcStr, dstStr, protoName, flags)
}

func IsChatter(packet gopacket.Packet) bool {
	switch layer := packet.NetworkLayer().(type) {
	case *layers.IPv4:
		// Note: directed broadcast (i.e. all ones in the host part) is not covered for now
		isBroadcast := layer.DstIP.Equal(net.IPv4bcast)
		isMulticast := layer.DstIP[0]&0xf0 == 0xe0
		return isBroadcast || isMulticast
	case *layers.IPv6:
		return layer.DstIP[0] == 0xff
	default:
		return false
	}
}
