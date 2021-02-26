package bizarre_net

// IPv4 messages start with 0x4 and IPv6 messages start with 0x6. These don't need a specific "header" byte.

var HELLO_MESSAGE = []byte{0x01, 0x00}
var HELLO_ACK_MESSAGE = []byte{0x01, 0x01}
