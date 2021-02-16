package main

import (
	"github.com/CapacitorSet/bizarre-net/lib/server"
	"log"
)

func main() {
	err := server.Run("config.toml", nil)
	if err != nil {
		log.Fatal(err)
	}
}
