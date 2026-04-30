package main

import (
	"fmt"
	"log"
	"net/http"

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

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Northgate Stores SRMS is running")
	})

	addr := ":8080"

	log.Printf("Starting server on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
