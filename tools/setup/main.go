package main

import (
	"bytes"
	"fmt"
	"github.com/CapacitorSet/bizarre-net/tools"
	"github.com/fatih/color"
	"os"
)

var grey = color.New(color.FgWhite)
var boldWhite = color.New(color.FgHiWhite, color.Bold).SprintFunc()

func NSCreate() {
	output := tools.IpExec("netns")
	if bytes.Contains(output, []byte("srvns")) {
		// color.White("srvns is already present.")
		color.Yellow("The environment seems to be already set up. If it isn't, try tearing it down and setting it up again.")
		os.Exit(1)
	} else {
		fmt.Println("Creating server namespace...")
		tools.IpExec("netns add srvns")
		tools.IpExecNetns("srvns", "link set lo up")
	}
	if bytes.Contains(output, []byte("clins")) {
		// color.White("clins is already present.")
		color.Yellow("The environment seems to be already set up. If it isn't, try tearing it down and setting it up again.")
		os.Exit(1)
	} else {
		fmt.Println("Creating client namespace...")
		tools.IpExec("netns add clins")
		tools.IpExecNetns("clins", "link set lo up")
	}
}

func VethCreate() {
	fmt.Println("Creating client-server veth...")
	tools.IpExec("link add seth0 type veth peer name ceth0")
	tools.IpExec("link set seth0 netns srvns")
	tools.IpExec("link set ceth0 netns clins")
	fmt.Printf("Server: configuring %s with IP %s...\n", boldWhite("seth0"), boldWhite("192.168.1.1"))
	tools.IpExecNetns("srvns", "link set seth0 up")
	tools.IpExecNetns("srvns", "addr add 192.168.1.1/24 dev seth0")
	fmt.Printf("Client: configuring %s with IP %s...\n", boldWhite("ceth0"), boldWhite("192.168.1.2"))
	tools.IpExecNetns("clins", "link set ceth0 up")
	tools.IpExecNetns("clins", "addr add 192.168.1.2/24 dev ceth0")

	fmt.Println("Creating server-host veth...")
	tools.IpExec("link add seth1 type veth peer name heth1")
	tools.IpExec("link set seth1 netns srvns")
	fmt.Printf("Server: configuring %s with IP %s...\n", boldWhite("seth1"), boldWhite("192.168.2.1"))
	tools.IpExecNetns("srvns", "link set seth1 up")
	tools.IpExecNetns("srvns", "addr add 192.168.2.1/24 dev seth1")
	fmt.Printf("Host: configuring %s with IP %s...\n", boldWhite("heth1"), boldWhite("192.168.2.3"))
	tools.IpExec("link set heth1 up")
	tools.IpExec("addr add 192.168.2.3/24 dev heth1")
	fmt.Printf("Server: configuring default route towards %s...\n", boldWhite("192.168.2.3"))
	tools.IpExecNetns("srvns", "route add default via 192.168.2.3")
}

func main() {
	tools.RootCheck()
	grey.EnableColor()
	NSCreate()
	VethCreate()
	color.Green("Done!")
}
