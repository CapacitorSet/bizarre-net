package bizarre_net

import (
	"bytes"
)

// IPv4 messages start with 0x4 and IPv6 messages start with 0x6. These don't need a specific "header" byte.

func TryParseHello(buffer []byte) []byte {
	if bytes.HasPrefix(buffer, HELLO_PREFIX) {
		return buffer[len(HELLO_PREFIX):]
	} else {
		return nil
	}
}

func TryParseHelloAck(buffer []byte) bool {
	if bytes.Equal(buffer, HELLO_ACK_MESSAGE) {
		return true
	} else {
		return false
	}
}

var HELLO_PREFIX = []byte{0x01, 0x00}
var HELLO_ACK_MESSAGE = []byte{0x01, 0x01}
