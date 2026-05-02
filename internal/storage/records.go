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

func UpdateEmployeeContactFields(db *sql.DB, userID int64, phone string, emergencyContact string) error {
	result, err := db.Exec(
		`UPDATE employee_records
		 SET phone = ?,
		     emergency_contact = ?,
		     last_updated_by = ?,
		     last_updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = ?`,
		phone,
		emergencyContact,
		userID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update employee contact fields: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated employee contact rows: %w", err)
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func GetAllEmployeeRecords(db *sql.DB) ([]EmployeeRecord, error) {
	rows, err := db.Query(
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
		ORDER BY last_name, first_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("get all employee records: %w", err)
	}
	defer rows.Close()

	var records []EmployeeRecord

	for rows.Next() {
		var record EmployeeRecord

		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan employee record: %w", err)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate employee records: %w", err)
	}
	return records, nil
}

func GetEmployeeRecordByID(db *sql.DB, recordID int64) (EmployeeRecord, error) {
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
		WHERE id = ?`,
		recordID,
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
		return EmployeeRecord{}, fmt.Errorf("get employee record by id: %w", err)
	}
	return record, nil
}
