package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	service := NewCustomerService(NewMemoryCustomerRepository(), NewMemoryOperationLogRepository())
	server := NewHTTPServer(service)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("medical companion service listening on http://127.0.0.1:%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatal(err)
	}
}
