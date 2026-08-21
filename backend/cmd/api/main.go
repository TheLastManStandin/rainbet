package main

import (
	"log"
	"net/http"
	"os"

	"rainbet/internal/application"
	"rainbet/internal/delivery/httpapi"
	"rainbet/internal/infrastructure/fairness"
	"rainbet/internal/infrastructure/password"
	"rainbet/internal/infrastructure/sqlite"
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

	db, err := sqlite.Open(databasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	store := sqlite.NewStore(db)
	accounts := application.NewAccountService(store.Accounts(), password.Bcrypt{})
	games := application.NewMinesService(store, fairness.Generator{})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: httpapi.New(accounts, games),
	}

	log.Printf("HTTP server listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
