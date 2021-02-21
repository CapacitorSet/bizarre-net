package generic

import (
	"bytes"
	"fmt"
	"github.com/CapacitorSet/bizarre-net/lib/client"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"testing"
	"time"
)

const successString = "Hello from bizarre-net!\n"

func waitErrChan(t *testing.T, ch chan *rpc.Call) {
	select {
	case err := <-ch:
		if err.Error != nil {
			fmt.Println("\n=*= SERVER ERROR FOLLOWS =*=")
			panic(err.Error)
		}
	}
}

type HostConfig struct {
	Config string
	TunIP  string
	VethIP string
}

type TestConfig struct {
	Client, Server HostConfig
}

type EmptyArgs struct{}

func (T TestConfig) ClientTest(t *testing.T) {
	// Launch client
	clientConfigFile, err := ioutil.TempFile("", "ClientTest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Remove(clientConfigFile.Name())
	})
	_, err = clientConfigFile.Write([]byte(T.Client.Config))
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Creating client")
	client, err := client.NewClient(clientConfigFile.Name(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create server
	log.Println("Dialing RPC server")
	server, err := rpc.DialHTTP("tcp", net.JoinHostPort(T.Server.VethIP, "1917"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var reply error
		server.Call("Server.RPCStop", EmptyArgs{}, &reply)
	})
	var reply error
	log.Println("Creating server")
	err = server.Call("Server.New", EmptyArgs{}, &reply)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Launching server")
	udpRunDoneChan := make(chan *rpc.Call, 1)
	server.Go("Server.Run", EmptyArgs{}, &reply, udpRunDoneChan)
	go waitErrChan(t, udpRunDoneChan)
	time.Sleep(time.Second / 2)

	log.Println("Launching client")
	go func() {
		err := client.Run()
		if err != nil {
			t.Fatal(err)
		}
	}()

	log.Println("Launching HTTP server")
	httpServeDoneChan := make(chan *rpc.Call, 1)
	server.Go("Server.HTTPServe", EmptyArgs{}, &reply, httpServeDoneChan)
	go waitErrChan(t, httpServeDoneChan)
	time.Sleep(time.Second / 2)

	log.Println("Making HTTP request")
	resp, err := http.Get("http://" + net.JoinHostPort(T.Server.TunIP, "2021") + "/hello")
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
