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
	b := newBackend()

	addr := envOr("PORT", "8080")
	log.Printf("cal (headless) on :%s", addr)

	// A failed init should fail the process, not just the status endpoint —
	// CI and supervisors rely on a nonzero exit.
	go func() {
		<-b.ready
		if b.handler == nil {
			log.Fatalf("cal: %s", b.status.Error)
		}
	}()

	log.Fatal(http.ListenAndServe(":"+addr, b))
}
