package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

func RenderTemplate(w http.ResponseWriter, name string, data any) {
	templatePath := filepath.Join("templates", name)

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Println("template execution error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
