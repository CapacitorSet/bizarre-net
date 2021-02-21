package socket

import (
	"github.com/CapacitorSet/bizarre-net/test/generic"
	"testing"
)

const clientConfig = `Transport = "socket"

[tun]
Prefix = "testbizarre"
IP = "20.20.20.1/24"
SetDefaultGW = false

[socket]
Socket = "/tmp/bizarre-test.sock"`

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
