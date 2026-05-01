package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrRecordNotFound = errors.New("employee record not found")

type EmployeeRecord struct {
	ID                 int64
	UserID             int64
	FirstName          string
	LastName           string
	Email              string
	Phone              string
	Address            string
	EmergencyContact   string
	Department         string
	JobTitle           string
	EmploymentStatus   string
	SalaryBand         string
	AccessibilityNotes string
	PrivateHRNotes     string
	LastUpdatedBy      int64
	LastUpdatedAt      string
}

func GetEmployeeRecordByUserID(db *sql.DB, userID int64) (EmployeeRecord, error) {
	var record EmployeeRecord

	err := db.QueryRow(
		`SELECT
			id,
			user_id,
			first_name,
			last_name,
			email,
			phone,
			address,
			emergency_contact,
			department,
			job_title,
			employment_status,
			salary_band,
			COALESCE(accessibility_notes, ''),
			COALESCE(private_hr_notes, ''),
			last_updated_by,
			last_updated_at
		FROM employee_records
		WHERE user_id = ?`,
		userID,
	).Scan(
		&record.ID,
		&record.UserID,
		&record.FirstName,
		&record.LastName,
		&record.Email,
		&record.Phone,
		&record.Address,
		&record.EmergencyContact,
		&record.Department,
		&record.JobTitle,
		&record.EmploymentStatus,
		&record.SalaryBand,
		&record.AccessibilityNotes,
		&record.PrivateHRNotes,
		&record.LastUpdatedBy,
		&record.LastUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmployeeRecord{}, ErrRecordNotFound
		}
		return EmployeeRecord{}, fmt.Errorf("get employee record by user id: %w", err)
	}
	return record, nil
}
