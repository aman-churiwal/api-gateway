package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

var (
	// Control failure mode via atomic to be goroutine-safe
	failMode atomic.Bool
	// Count requests for debugging
	requestCount atomic.Int64
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		log.Printf("[%d] Received request: %s %s", count, r.Method, r.URL.Path)

		// Check for control endpoints
		switch r.URL.Path {
		case "/control/fail":
			failMode.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"mode": "fail", "message": "Backend will now return 500 errors"}`)
			log.Println("MODE: Switched to FAIL mode")
			return

		case "/control/recover":
			failMode.Store(false)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"mode": "ok", "message": "Backend will now return 200 OK"}`)
			log.Println("MODE: Switched to OK mode")
			return

		case "/control/status":
			w.Header().Set("Content-Type", "application/json")
			mode := "ok"
			if failMode.Load() {
				mode = "fail"
			}
			fmt.Fprintf(w, `{"mode": "%s", "request_count": %d}`, mode, count)
			return
		}

		// Check if in fail mode
		if failMode.Load() {
			log.Printf("[%d] Responding with 500 (fail mode)", count)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "Simulated backend failure", "request": %d}`, count)
			return
		}

		// Normal response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "Hello from dummy backend on port 3001", "path": "%s", "request": %d}`, r.URL.Path, count)
	})

	log.Println("Dummy backend starting on :3001")
	log.Println("Control endpoints:")
	log.Println("  GET /control/fail    - Enable 500 errors")
	log.Println("  GET /control/recover - Return to normal")
	log.Println("  GET /control/status  - Check current mode")

	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}
}
