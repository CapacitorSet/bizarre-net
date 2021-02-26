package udp

import (
	"github.com/CapacitorSet/bizarre-net/test/generic"
	"testing"
)

const clientConfig = `Transport = "udp"
SendHello = false

[tun]
Prefix = "testbizarre"
IP = "20.20.20.1/24"
SetDefaultGW = false

[udp]
IP = "192.168.1.2:1917"`

var testConfig = generic.TestConfig{
	Client: generic.HostConfig{
		Config: clientConfig,
		TunIP:  "20.20.20.1",
		VethIP: "192.168.1.3",
	},
	Server: generic.HostConfig{
		Config: serverConfig,
		TunIP:  "20.20.20.2",
		VethIP: "192.168.1.2",
	},
}

func TestClient(t *testing.T) {
	testConfig.ClientTest(t)
}
