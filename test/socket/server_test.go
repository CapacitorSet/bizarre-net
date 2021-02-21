package socket

import "testing"

const serverConfig = `Transport = "socket"

[tun]
Prefix = "testbizarre"
IP = "20.20.20.2/24"

[socket]
Socket = "/tmp/bizarre-test.sock"`

func TestServer(t *testing.T) {
	testConfig.ServerTest(t)
}
