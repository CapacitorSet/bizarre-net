package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CapacitorSet/bizarre-net/lib/client"
)

func main() {
	flagset := flag.NewFlagSet("", flag.ExitOnError)
	help := flagset.Bool("help", false, "Show usage information")
	clientConf := client.NewConfigFromFlags(flagset)
	flagset.Parse(os.Args[1:])

	if (*help) {
		flagset.PrintDefaults()
		return
	}

	srv, err := client.NewClient(clientConf)
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
