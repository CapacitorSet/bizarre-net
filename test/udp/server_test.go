package udp

import (
	"fmt"
	"github.com/CapacitorSet/bizarre-net/lib/server"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"testing"
)

const serverConfig = `Transport = "udp"

[tun]
Prefix = "testbizarre"
IP = "20.20.20.2/24"

[udp]
IP = "0.0.0.0:1917"`

type UDPServer struct {
	*testing.T
	server.Server
	doneChan chan bool
}

type EmptyArgs struct{}

func (S *UDPServer) New(args *EmptyArgs, reply *error) error {
	serverConfigFile, err := ioutil.TempFile("", "TestUDP")
	if err != nil {
		S.Error(err)
		return err
	}
	defer os.Remove(serverConfigFile.Name())
	_, err = serverConfigFile.Write([]byte(serverConfig))
	if err != nil {
		S.Error(err)
		return err
	}
	srv, err := server.NewServer(serverConfigFile.Name(), nil)
	if err != nil {
		S.Error(err)
		return err
	}
	S.Server = srv
	return nil
}

func (S UDPServer) Run(args *EmptyArgs, reply *error) error {
	err := S.Server.Run()
	if err != nil {
		S.Error(err)
		return err
	}
	return nil
}

func (S UDPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var response string
	if r.URL.Path == "/hello" {
		response = successString
	} else {
		response = fmt.Sprintf("Expected /hello, got %s", r.URL.Path)
	}
	_, err := w.Write([]byte(response))
	if err != nil {
		S.Error(err)
	}
}

func (S UDPServer) HTTPServe(args *EmptyArgs, reply *error) error {
	err := http.ListenAndServe("20.20.20.2:2021", S)
	if err != nil {
		S.Error(err)
		return err
	}
	return nil
}

func (S UDPServer) RPCStop(args *EmptyArgs, reply *error) error {
	S.doneChan <- true
	return nil
}

func TestUDPServer(t *testing.T) {
	server := &UDPServer{T: t, Server: nil, doneChan: make(chan bool, 1)}
	err := rpc.Register(server)
	if err != nil {
		t.Fatal(err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", "0.0.0.0:1917")
	if err != nil {
		t.Fatal(err)
	}
	log.Println("RPC server up")
	go http.Serve(l, nil)
	select {
	case <-server.doneChan:
		return
	}
}
