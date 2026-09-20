package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/httpserver"
)

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":18080"
	}
	timeout := 8 * time.Second
	if v := os.Getenv("HTTP_TIMEOUT_MS"); v != "" {
		if n, err := time.ParseDuration(v + "ms"); err == nil {
			timeout = n
		}
	}
	srv := httpserver.New(timeout)
	log.Printf("api-go listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
