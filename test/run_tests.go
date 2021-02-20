package main

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

func main() {
	serverTestCmd := exec.Command("nsenter", "--net=/var/run/netns/srvns", "go", "test", "-run", "TestUDPServer")
	serverTestCmd.Dir = "test/udp"
	serverTestCmd.Stdout = os.Stdout
	serverTestCmd.Stderr = os.Stderr
	clientTestCmd := exec.Command("nsenter", "--net=/var/run/netns/clins", "go", "test", "-run", "TestUDPClient")
	clientTestCmd.Dir = "test/udp"
	clientTestCmd.Stdout = os.Stdout
	clientTestCmd.Stderr = os.Stderr
	wg := sync.WaitGroup{}
	wg.Add(2)
	finalExitCode := 0
	exitCodes := make(chan int, 1)
	go func() {
		select {
		case code := <-exitCodes:
			if code != 0 {
				finalExitCode = code
			}
		}
	}()
	go func() {
		err := serverTestCmd.Run()
		if err != nil {
			log.Printf("Server test: %s\n", err.Error())
		}
		exitCodes <- serverTestCmd.ProcessState.ExitCode()
		wg.Done()
	}()
	go func() {
		err := clientTestCmd.Run()
		if err != nil {
			log.Printf("Client test: %s\n", err.Error())
		}
		exitCodes <- clientTestCmd.ProcessState.ExitCode()
		wg.Done()
	}()
	wg.Wait()
	os.Exit(finalExitCode)
}
