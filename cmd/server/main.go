package main

import (
	"log"
	"net/http"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/config"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/handlers"
	"northgate-srms/internal/middleware"
	"northgate-srms/internal/security"
	"northgate-srms/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	db, err := storage.OpenDatabase(cfg.DBPath)
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
	csrfManager := csrf.NewManager()
	loginLimiter := security.NewLoginLimiter()

	authHandler := handlers.NewAuthHandler(db, sessionManager, csrfManager, loginLimiter)
	homeHandler := handlers.NewHomeHandler(sessionManager, csrfManager)
	recordHandler := handlers.NewRecordHandler(db, sessionManager, csrfManager)
	adminHandler := handlers.NewAdminHandler(db, sessionManager, csrfManager)

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

	addr := cfg.ServerAddress()

	log.Printf("Starting server on http://localhost%s", addr)

	handler := middleware.SecurityHeaders(mux)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
