package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/storage"
	"northgate-srms/internal/validation"
)

type RecordHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
	CSRF     *csrf.Manager
}

type RecordPageData struct {
	Username  string
	Role      string
	Record    storage.EmployeeRecord
	CSRFToken string
}

type RecordEditPageData struct {
	Username  string
	Role      string
	Record    storage.EmployeeRecord
	Error     string
	CSRFToken string
}

func NewRecordHandler(db *sql.DB, sessions *auth.SessionManager, csrfManager *csrf.Manager) *RecordHandler {
	return &RecordHandler{
		DB:       db,
		Sessions: sessions,
		CSRF:     csrfManager,
	}
}

func (h *RecordHandler) ViewOwnRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.Sessions.Get(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	token, err := h.CSRF.Generate(session.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
		Username:  session.User.Username,
		Role:      session.User.Role,
		Record:    record,
		CSRFToken: token,
	}

	RenderTemplate(w, "record.html", data)
}

func (h *RecordHandler) EditOwnRecord(w http.ResponseWriter, r *http.Request) {
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
	token, err := h.CSRF.Generate(session.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := RecordEditPageData{
		Username:  session.User.Username,
		Role:      session.User.Role,
		Record:    record,
		CSRFToken: token,
	}

	RenderTemplate(w, "record_edit.html", data)
}

func (h *RecordHandler) UpdateOwnRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.Sessions.Get(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !h.CSRF.Validate(r, session.ID) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	phone := r.FormValue("phone")
	emergencyContact := r.FormValue("emergency_contact")

	if phone == "" || emergencyContact == "" {
		h.renderEditFormWithError(w, session, "Phone and emergency contact are required.")
		return
	}

	if !validation.IsValidPhone(phone) {
		h.renderEditFormWithError(w, session, "Phone must be 7–20 characters and contain only numbers, spaces, plus signs, or hyphens")
		return
	}

	if !validation.IsValidEmergencyContact(emergencyContact) {
		h.renderEditFormWithError(w, session, "Emergency contact must be between 5 and 120 characters.")
		return
	}

	if err := storage.UpdateEmployeeContactFields(h.DB, session.User.ID, phone, emergencyContact); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/record", http.StatusSeeOther)
}

func (h *RecordHandler) renderEditFormWithError(w http.ResponseWriter, session auth.Session, message string) {
	record, err := storage.GetEmployeeRecordByUserID(h.DB, session.User.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := h.CSRF.Generate(session.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := RecordEditPageData{
		Username:  session.User.Username,
		Role:      session.User.Role,
		Record:    record,
		Error:     message,
		CSRFToken: token,
	}

	RenderTemplate(w, "record_edit.html", data)
}
