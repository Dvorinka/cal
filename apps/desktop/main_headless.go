//go:build headless

// Headless build: serves the same handler over TCP so the desktop bundle can
// be exercised without webkit/wails (CI, servers, dev boxes).
//
//	go build -tags headless -o cal . && ./cal
package main

import (
	"log"
	"net/http"
)

func main() {
	handler, closeDB, err := NewHandler()
	if err != nil {
		log.Fatalf("cal: %v", err)
	}
	defer closeDB()
	addr := envOr("PORT", "8080")
	log.Printf("cal (headless) on :%s", addr)
	log.Fatal(http.ListenAndServe(":"+addr, handler))
}
