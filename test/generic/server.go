package generic

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

type Server struct {
	TestConfig
	*testing.T
	server.Server
	doneChan chan bool
}

func (S *Server) New(args *EmptyArgs, reply *error) error {
	serverConfigFile, err := ioutil.TempFile("", "TestServer")
	if err != nil {
		S.Error(err)
		return err
	}
	defer os.Remove(serverConfigFile.Name())
	_, err = serverConfigFile.Write([]byte(S.TestConfig.Server.Config))
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

func (S Server) Run(args *EmptyArgs, reply *error) error {
	err := S.Server.Run()
	if err != nil {
		S.Error(err)
		return err
	}
	return nil
}

func (S Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (S Server) HTTPServe(args *EmptyArgs, reply *error) error {
	err := http.ListenAndServe(net.JoinHostPort(S.TestConfig.Server.TunIP, "2021"), S)
	if err != nil {
		S.Error(err)
		return err
	}
	return nil
}

func (S Server) RPCStop(args *EmptyArgs, reply *error) error {
	S.doneChan <- true
	return nil
}

func (T TestConfig) ServerTest(t *testing.T) {
	server := &Server{TestConfig: T, T: t, Server: nil, doneChan: make(chan bool, 1)}
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
