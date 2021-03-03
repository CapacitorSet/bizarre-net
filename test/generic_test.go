package main

import (
	"os"
	"os/exec"
	"sync"
	"testing"
)

func testServer(t *testing.T, cmdDir string) {
	serverTestCmd := exec.Command("nsenter", "--net=/var/run/netns/srvns", "go", "test", "-run", "TestServer")
	serverTestCmd.Dir = cmdDir
	serverTestCmd.Stdout = os.Stdout
	serverTestCmd.Stderr = os.Stderr
	clientTestCmd := exec.Command("nsenter", "--net=/var/run/netns/clins", "go", "test", "-run", "TestClient")
	clientTestCmd.Dir = cmdDir
	clientTestCmd.Stdout = os.Stdout
	clientTestCmd.Stderr = os.Stderr
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		err := serverTestCmd.Run()
		if err != nil {
			t.Error(err)
		}
		wg.Done()
	}()
	go func() {
		err := clientTestCmd.Run()
		if err != nil {
			t.Error(err)
		}
		wg.Done()
	}()
	wg.Wait()
}

func TestUDP(t *testing.T) {
	testServer(t, "udp")
}

