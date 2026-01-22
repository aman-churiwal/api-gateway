package main

import (
	"flag"
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
	// Port the backend is running on
	port string
)

func main() {
	flag.StringVar(&port, "port", "3001", "Port to listen on")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		log.Printf("[%d] Received request: %s %s", count, r.Method, r.URL.Path)

		// Check for control endpoints
		switch r.URL.Path {
		case "/health":
			// Health check endpoint for health checker
			if failMode.Load() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, `{"status": "unhealthy", "port": "%s"}`, port)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status": "healthy", "port": "%s"}`, port)
			return

		case "/control/fail":
			failMode.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"mode": "fail", "message": "Backend will now return 500 errors", "port": "%s"}`, port)
			log.Println("MODE: Switched to FAIL mode")
			return

		case "/control/recover":
			failMode.Store(false)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"mode": "ok", "message": "Backend will now return 200 OK", "port": "%s"}`, port)
			log.Println("MODE: Switched to OK mode")
			return

		case "/control/status":
			w.Header().Set("Content-Type", "application/json")
			mode := "ok"
			if failMode.Load() {
				mode = "fail"
			}
			fmt.Fprintf(w, `{"mode": "%s", "request_count": %d, "port": "%s"}`, mode, count, port)
			return
		}

		// Check if in fail mode
		if failMode.Load() {
			log.Printf("[%d] Responding with 500 (fail mode)", count)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "Simulated backend failure", "request": %d, "port": "%s"}`, count, port)
			return
		}

		// Normal response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "Hello from dummy backend", "port": "%s", "path": "%s", "request": %d}`, port, r.URL.Path, count)
	})

	addr := ":" + port
	log.Printf("Dummy backend starting on %s", addr)
	log.Println("Endpoints:")
	log.Println("  GET /health          - Health check endpoint")
	log.Println("  GET /control/fail    - Enable 500 errors")
	log.Println("  GET /control/recover - Return to normal")
	log.Println("  GET /control/status  - Check current mode")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
