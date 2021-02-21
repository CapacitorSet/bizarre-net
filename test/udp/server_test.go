package udp

import (
	"testing"
)

const serverConfig = `Transport = "udp"

[tun]
Prefix = "testbizarre"
IP = "20.20.20.2/24"

[udp]
IP = "0.0.0.0:1917"`

func TestServer(t *testing.T) {
	testConfig.ServerTest(t)
}
