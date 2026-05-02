package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"northgate-srms/internal/auth"
	"northgate-srms/internal/storage"
	"strconv"
)

type AdminHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
}

type AdminRecordsPageData struct {
	Username string
	Role     string
	Records  []storage.EmployeeRecord
}

type AdminRecordViewPageData struct {
	Username string
	Role     string
	Record   storage.EmployeeRecord
}

func NewAdminHandler(db *sql.DB, sessions *auth.SessionManager) *AdminHandler {
	return &AdminHandler{
		DB:       db,
		Sessions: sessions,
	}
}

func (h *AdminHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	records, err := storage.GetAllEmployeeRecords(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := AdminRecordsPageData{
		Username: session.User.Username,
		Role:     session.User.Role,
		Records:  records,
	}

	RenderTemplate(w, "admin_records.html", data)
}

func (h *AdminHandler) ViewRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	recordID, err := parseRecordID(r)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record, err := storage.GetEmployeeRecordByID(h.DB, recordID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			http.Error(w, "Employee record not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := AdminRecordViewPageData{
		Username: session.User.Username,
		Role:     session.User.Role,
		Record:   record,
	}

	RenderTemplate(w, "admin_record_view.html", data)
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	session, ok := h.Sessions.Get(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return auth.Session{}, false
	}

	if session.User.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return auth.Session{}, false
	}
	return session, true
}

func parseRecordID(r *http.Request) (int64, error) {
	rawID := r.URL.Query().Get("id")
	if rawID == "" {
		return 0, errors.New("missing record id")
	}

	recordID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || recordID <= 0 {
		return 0, errors.New("invalid record id")
	}
	return recordID, nil
}
