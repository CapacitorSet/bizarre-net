package udp

import (
	"bytes"
	"github.com/CapacitorSet/bizarre-net/lib/client"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"runtime/debug"
	"testing"
	"time"
)

const clientConfig = `Transport = "udp"

[tun]
Prefix = "testbizarre"
IP = "20.20.20.1/24"
SetDefaultGW = false

[udp]
IP = "192.168.1.2:1917"`

const successString = "Hello from bizarre-net!\n"

func waitErrChan(t *testing.T, ch chan *rpc.Call) {
	select {
	case err := <-ch:
		if err.Error != nil {
			debug.PrintStack()
			t.Fatal(err.Error)
		}
	}
}

func TestUDPClient(t *testing.T) {
	// Create UDP server
	log.Println("Dialing RPC server")
	server, err := rpc.DialHTTP("tcp", "192.168.1.2:1917")
	if err != nil {
		t.Fatal(err)
	}
	var reply error
	log.Println("Creating server")
	err = server.Call("UDPServer.New", EmptyArgs{}, &reply)
	if err != nil {
		t.Fatal(err)
	}

	// Launch UDP client
	clientConfigFile, err := ioutil.TempFile("", "TestUDP")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Remove(clientConfigFile.Name())
		var reply error
		server.Call("UDPServer.RPCStop", EmptyArgs{}, &reply)
	})
	_, err = clientConfigFile.Write([]byte(clientConfig))
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Creating client")
	client, err := client.NewClient(clientConfigFile.Name(), nil)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Launching UDP server")
	udpRunDoneChan := make(chan *rpc.Call, 1)
	server.Go("UDPServer.Run", EmptyArgs{}, &reply, udpRunDoneChan)
	go waitErrChan(t, udpRunDoneChan)

	log.Println("Launching client")
	go func() {
		err := client.Run()
		if err != nil {
			t.Fatal(err)
		}
	}()

	log.Println("Launching HTTP server")
	httpServeDoneChan := make(chan *rpc.Call, 1)
	server.Go("UDPServer.HTTPServe", EmptyArgs{}, &reply, httpServeDoneChan)
	go waitErrChan(t, httpServeDoneChan)
	time.Sleep(time.Second / 2)

	log.Println("Making HTTP request")
	resp, err := http.Get("http://20.20.20.2:2021/hello")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("HTTP response: %s\n", body)
	if !bytes.Equal(body, []byte(successString)) {
		t.Fatalf("Expected success string, got %#v", body)
	}
}
