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
	recordHandler := handlers.NewRecordHandler(db, sessionManager)
	adminHandler := handlers.NewAdminHandler(db, sessionManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler.Home)
	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	mux.HandleFunc("GET /record", recordHandler.ViewOwnRecord)
	mux.HandleFunc("GET /record/edit", recordHandler.EditOwnRecord)
	mux.HandleFunc("POST /record/update", recordHandler.UpdateOwnRecord)

	mux.HandleFunc("GET /admin/records", adminHandler.ListRecords)
	mux.HandleFunc("GET /admin/records/view", adminHandler.ViewRecord)
	mux.HandleFunc("GET /admin/records/edit", adminHandler.EditRecord)
	mux.HandleFunc("POST /admin/records/update", adminHandler.UpdateRecord)

	addr := ":8080"

	log.Printf("Starting server on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
