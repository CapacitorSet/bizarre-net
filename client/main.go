package main

import (
	"github.com/CapacitorSet/bizarre-net/lib/client"
	"log"
)

func main() {
	err := client.Run("config.toml", nil)
	if err != nil {
		log.Fatal(err)
	}
}
