package main

import (
	"github.com/CapacitorSet/bizarre-net/lib/client"
	"log"
)

func main() {
	client, err := client.NewClient("config.toml", nil)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Run()
	if err != nil {
		log.Fatal(err)
	}
}
