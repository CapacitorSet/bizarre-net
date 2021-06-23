package tools

import (
	"fmt"
	"github.com/fatih/color"
	"os/exec"
	"os/user"
	"strings"
)

func RootCheck() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if u.Uid == "0" {
		fmt.Println("You are root.")
	} else {
		color.Yellow("[!] You aren't root; you probably need it.")
	}
}

func IpExec(args string) []byte {
	output, err := exec.Command("ip", strings.Split(args, " ")...).Output()
	if err != nil {
		panic(err)
	}
	return output
}

// Execute "ip" + args in the given network namespace
func IpExecNetns(netns string, args string) []byte {
	nsenterArgs := []string{"--net=/var/run/netns/" + netns, "ip"}
	nsenterArgs = append(nsenterArgs, strings.Split(args, " ")...)
	output, err := exec.Command("nsenter", nsenterArgs...).Output()
	if err != nil {
		panic(err)
	}
	return output
}

