package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"northgate-srms/internal/auth"
	"northgate-srms/internal/storage"
)

type RecordHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
}

type RecordPageData struct {
	Username string
	Role     string
	Record   storage.EmployeeRecord
}

func NewRecordHandler(db *sql.DB, sessions *auth.SessionManager) *RecordHandler {
	return &RecordHandler{
		DB:       db,
		Sessions: sessions,
	}
}

func (h *RecordHandler) ViewOwnRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.Sessions.Get(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	record, err := storage.GetEmployeeRecordByUserID(h.DB, session.User.ID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			http.Error(w, "Employee record not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := RecordPageData{
		Username: session.User.Username,
		Role:     session.User.Role,
		Record:   record,
	}

	RenderTemplate(w, "record.html", data)
}
