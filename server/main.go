package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CapacitorSet/bizarre-net/lib/server"
)

func main() {
	flagset := flag.NewFlagSet("", flag.ExitOnError)
	help := flagset.Bool("help", false, "Show usage information")
	serverConf := server.NewConfigFromFlags(flagset)
	flagset.Parse(os.Args[1:])

	if (*help) {
		flagset.PrintDefaults()
		return
	}

	srv, err := server.NewServer(serverConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = srv.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
