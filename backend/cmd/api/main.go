package main

import (
	"log"
	"net/http"
	"os"

	"rainbet/internal/router"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	username := os.Getenv("BASIC_AUTH_USERNAME")
	password := os.Getenv("BASIC_AUTH_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("BASIC_AUTH_USERNAME and BASIC_AUTH_PASSWORD must be set")
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router.New(username, password),
	}

	log.Printf("HTTP server listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
