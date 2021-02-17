package main

import (
	"github.com/CapacitorSet/bizarre-net/lib/server"
	"log"
)

func main() {
	srv, err := server.NewServer("config.toml", nil)
	if err != nil {
		log.Fatal(err)
	}
	err = srv.Run()
	if err != nil {
		log.Fatal(err)
	}
}
