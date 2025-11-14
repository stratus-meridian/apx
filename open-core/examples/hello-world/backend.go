package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", handleRequest)

	port := 9000
	fmt.Printf("Starting backend server on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message":   "Hello from APX Router Open-Core!",
		"path":      r.URL.Path,
		"method":    r.Method,
		"timestamp": time.Now().Format(time.RFC3339),
		"headers":   r.Header,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[Backend] %s %s", r.Method, r.URL.Path)
}
