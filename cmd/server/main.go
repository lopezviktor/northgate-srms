package main

import (
	"log"
	"net/http"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/handlers"
	"northgate-srms/internal/storage"
)

func main() {
	db, err := storage.OpenDatabase("northgate.db")
	if err != nil {
		log.Fatalf("database setup failed: %v", err)
	}
	defer db.Close()

	if err := storage.CreateSchema(db); err != nil {
		log.Fatalf("schema setup failed: %v", err)
	}

	if err := storage.SeedDemoData(db); err != nil {
		log.Fatalf("seed data failed: %v", err)
	}

	sessionManager := auth.NewSessionManager()
	authHandler := handlers.NewAuthHandler(db, sessionManager)
	homeHandler := handlers.NewHomeHandler(sessionManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler.Home)
	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", authHandler.Login)

	addr := ":8080"

	log.Printf("Starting server on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
