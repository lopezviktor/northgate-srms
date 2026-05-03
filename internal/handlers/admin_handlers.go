package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"northgate-srms/internal/auth"
	"northgate-srms/internal/csrf"
	"northgate-srms/internal/storage"
	"strconv"
)

type AdminHandler struct {
	DB       *sql.DB
	Sessions *auth.SessionManager
	CSRF     *csrf.Manager
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

	if hasEmptyRequiredAdminFields(record) {
		h.renderAdminEditFormWithError(w, session, record, "Required fields cannot be empty.")
		return
	}

	if hasTooLongAdminFields(record) {
		h.renderAdminEditFormWithError(w, session, record, "One or more fields are too long.")
		return
	}

	if !isAllowedDepartment(record.Department) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid department.")
		return
	}

	if !isAllowedEmploymentStatus(record.EmploymentStatus) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid employment status.")
		return
	}

	if !isAllowedSalaryBand(record.SalaryBand) {
		h.renderAdminEditFormWithError(w, session, record, "Invalid salary band.")
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

func hasEmptyRequiredAdminFields(record storage.EmployeeRecord) bool {
	return record.FirstName == "" ||
		record.LastName == "" ||
		record.Email == "" ||
		record.Phone == "" ||
		record.Address == "" ||
		record.EmergencyContact == "" ||
		record.Department == "" ||
		record.JobTitle == "" ||
		record.EmploymentStatus == "" ||
		record.SalaryBand == ""
}

func hasTooLongAdminFields(record storage.EmployeeRecord) bool {
	return len(record.FirstName) > 50 ||
		len(record.LastName) > 50 ||
		len(record.Email) > 120 ||
		len(record.Phone) > 20 ||
		len(record.Address) > 150 ||
		len(record.EmergencyContact) > 120 ||
		len(record.JobTitle) > 60 ||
		len(record.AccessibilityNotes) > 500 ||
		len(record.PrivateHRNotes) > 1000
}

func isAllowedDepartment(department string) bool {
	switch department {
	case "Sales", "Stockroom", "Management", "HR", "Operations":
		return true
	default:
		return false
	}
}

func isAllowedEmploymentStatus(status string) bool {
	switch status {
	case "active", "on_leave", "terminated":
		return true
	default:
		return false
	}
}

func isAllowedSalaryBand(band string) bool {
	switch band {
	case "A", "B", "C", "D", "E":
		return true
	default:
		return false
	}
}
