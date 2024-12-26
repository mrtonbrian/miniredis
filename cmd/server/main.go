package main

import (
	"log"

	"github.com/mrtonbrian/miniredis/internal/miniredis"
)

func main() {
	addr := "0.0.0.0:6379"
	log.Printf("Starting server on %s\n", addr)

	err := miniredis.StartServer(addr)
	if err != nil {
		log.Printf("Server Error: %v\n", err)
	}
}
