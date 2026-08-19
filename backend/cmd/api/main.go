package main

import (
	"log"
	"net/http"
	"os"

	"rainbet/internal/database"
	"rainbet/internal/router"
	"rainbet/internal/user"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "rainbet.db"
	}

	db, err := database.OpenSQLite(databasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router.New(user.NewStore(db)),
	}

	log.Printf("HTTP server listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
