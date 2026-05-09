package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/storage"
	"northgate-srms/internal/validation"
)

type AdminHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
	CSRF     *csrf.Manager
}

type AdminRecordsPageData struct {
	Username  string
	Role      string
	Records   []storage.EmployeeRecord
	CSRFToken string
}

type AdminRecordViewPageData struct {
	Username              string
	Role                  string
	Record                storage.EmployeeRecord
	LastUpdatedByUsername string
	CSRFToken             string
}

type AdminRecordEditPageData struct {
	Username  string
	Role      string
	Record    storage.EmployeeRecord
	Error     string
	CSRFToken string
}

func NewAdminHandler(db *sql.DB, sessions *auth.SessionManager, csrfManager *csrf.Manager) *AdminHandler {
	return &AdminHandler{
		DB:       db,
		Sessions: sessions,
		CSRF:     csrfManager,
	}
}

func (h *AdminHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	token, err := h.CSRF.Generate(session.ID)
	if err != nil {

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return

	}
	records, err := storage.GetAllEmployeeRecords(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := AdminRecordsPageData{
		Username:  session.User.Username,
		Role:      session.User.Role,
		Records:   records,
		CSRFToken: token,
	}

	RenderTemplate(w, "admin_records.html", data)
}

func (h *AdminHandler) ViewRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	token, err := h.CSRF.Generate(session.ID)
	if err != nil {

		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	updatedByUsername, err := storage.GetUsernameByID(h.DB, record.LastUpdatedBy)
	if err != nil {
		updatedByUsername = "unknown"
	}

	data := AdminRecordViewPageData{
		Username:              session.User.Username,
		Role:                  session.User.Role,
		Record:                record,
		LastUpdatedByUsername: updatedByUsername,
		CSRFToken:             token,
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

func (h *AdminHandler) EditRecord(w http.ResponseWriter, r *http.Request) {
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

	token, err := h.CSRF.Generate(session.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return

	}
	data := AdminRecordEditPageData{
		Username:  session.User.Username,
		Role:      session.User.Role,
		Record:    record,
		CSRFToken: token,
	}

	RenderTemplate(w, "admin_record_edit.html", data)
}

func (h *AdminHandler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if !h.CSRF.Validate(r, session.ID) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	recordID, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil || recordID <= 0 {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	record := storage.EmployeeRecord{
		ID:                 recordID,
		FirstName:          r.FormValue("first_name"),
		LastName:           r.FormValue("last_name"),
		Email:              r.FormValue("email"),
		Phone:              r.FormValue("phone"),
		Address:            r.FormValue("address"),
		EmergencyContact:   r.FormValue("emergency_contact"),
		Department:         r.FormValue("department"),
		JobTitle:           r.FormValue("job_title"),
		EmploymentStatus:   r.FormValue("employment_status"),
		SalaryBand:         r.FormValue("salary_band"),
		AccessibilityNotes: r.FormValue("accessibility_notes"),
		PrivateHRNotes:     r.FormValue("private_hr_notes"),
	}

	if !validation.IsValidName(record.FirstName) {
		h.renderAdminEditFormWithError(w, session, record, "First name must be between 2 and 50 characters.")
		return
	}

	if !validation.IsValidName(record.LastName) {
		h.renderAdminEditFormWithError(w, session, record, "Last name must be between 2 and 50 characters.")
		return
	}

	if !validation.IsValidEmail(record.Email) {
		h.renderAdminEditFormWithError(w, session, record, "Email must be a valid email address.")
		return
	}

	if !validation.IsValidPhone(record.Phone) {
		h.renderAdminEditFormWithError(w, session, record, "Phone must be 7–20 characters and contain only numbers, spaces, plus signs, or hyphens.")
		return
	}

	if !validation.IsValidAddress(record.Address) {
		h.renderAdminEditFormWithError(w, session, record, "Address must be between 5 and 150 characters.")
		return
	}

	if !validation.IsValidEmergencyContact(record.EmergencyContact) {
		h.renderAdminEditFormWithError(w, session, record, "Emergency contact must be between 5 and 120 characters.")
		return
	}

	if !validation.IsAllowedDepartment(record.Department) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid department.")
		return
	}

	if !validation.IsValidJobTitle(record.JobTitle) {
		h.renderAdminEditFormWithError(w, session, record, "Job title must be between 2 and 60 characters.")
		return
	}

	if !validation.IsAllowedEmploymentStatus(record.EmploymentStatus) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid employment status.")
		return
	}

	if !validation.IsAllowedSalaryBand(record.SalaryBand) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid salary band.")
		return
	}

	if !validation.IsValidAccessibilityNotes(record.AccessibilityNotes) {
		h.renderAdminEditFormWithError(w, session, record, "Accessibility notes must be 500 characters or fewer.")
		return
	}

	if !validation.IsValidPrivateHRNotes(record.PrivateHRNotes) {
		h.renderAdminEditFormWithError(w, session, record, "Private HR notes must be 1000 characters or fewer.")
		return
	}

	if err := storage.UpdateEmployeeRecordByAdmin(h.DB, record, session.User.ID); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			http.Error(w, "Employee record not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/records/view?id="+strconv.FormatInt(recordID, 10), http.StatusSeeOther)
}

func (h *AdminHandler) renderAdminEditFormWithError(w http.ResponseWriter, session auth.Session, record storage.EmployeeRecord, message string) {
	token, err := h.CSRF.Generate(session.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := AdminRecordEditPageData{
		Username:  session.User.Username,
		Role:      session.User.Role,
		Record:    record,
		Error:     message,
		CSRFToken: token,
	}

	RenderTemplate(w, "admin_record_edit.html", data)
}
