package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "Hello from dummy backend on port 3001", "path": "%s"}`, r.URL.Path)
	})

	log.Println("Dummy backend starting on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}
}
