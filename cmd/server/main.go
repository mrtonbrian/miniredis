package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/mrtonbrian/miniredis/internal/miniredis"
)

func main() {
	addr := "0.0.0.0:6379"
	log.Printf("Starting server on %s\n", addr)

	go func() {
		log.Println("Starting pprof on :6060...")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	err := miniredis.StartServer(addr)
	if err != nil {
		// log.Printf("Server Error: %v\n", err)
	}
}
