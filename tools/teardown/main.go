package main

import (
	"bytes"
	"github.com/CapacitorSet/bizarre-net/tools"
	"github.com/fatih/color"
)

func NSTearDown() {
	output := tools.IpExec("netns")
	if bytes.Contains(output, []byte("srvns")) {
		color.White("Removing server namespace...")
		tools.IpExec("netns del srvns")
	} else {
		color.White("srvns does not exist.")
	}
	if bytes.Contains(output, []byte("clins")) {
		color.White("Removing client namespace...")
		tools.IpExec("netns del clins")
	} else {
		color.White("clins does not exist.")
	}
}

func main() {
	tools.RootCheck()
	NSTearDown()
	color.Green("Done!")
}
