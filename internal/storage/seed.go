package storage

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	Username string
	Password string
	Role     string
	Record   seedEmployeeRecord
}

type seedEmployeeRecord struct {
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
}

func SeedDemoData(db *sql.DB) error {
	users := []seedUser{
		{
			Username: "admin",
			Password: "AdminPass123!",
			Role:     "admin",
			Record: seedEmployeeRecord{
				FirstName:          "Helen",
				LastName:           "Carter",
				Email:              "helen.carter@northgate.test",
				Phone:              "+44 7700 900001",
				Address:            "1 Northgate Street, York",
				EmergencyContact:   "Mark Carter, +44 7700 900002",
				Department:         "HR",
				JobTitle:           "HR Administrator",
				EmploymentStatus:   "active",
				SalaryBand:         "D",
				AccessibilityNotes: "",
				PrivateHRNotes:     "Administrator account for assessment testing.",
			},
		},
		{
			Username: "alice",
			Password: "AlicePass123!",
			Role:     "employee",
			Record: seedEmployeeRecord{
				FirstName:          "Alice",
				LastName:           "Morgan",
				Email:              "alice.morgan@northgate.test",
				Phone:              "+44 7700 900101",
				Address:            "14 Foss Road, York",
				EmergencyContact:   "Daniel Morgan, +44 7700 900102",
				Department:         "Sales",
				JobTitle:           "Sales Assistant",
				EmploymentStatus:   "active",
				SalaryBand:         "B",
				AccessibilityNotes: "",
				PrivateHRNotes:     "Part-time contract. No active disciplinary notes.",
			},
		},
		{
			Username: "bob",
			Password: "BobPass123!",
			Role:     "employee",
			Record: seedEmployeeRecord{
				FirstName:          "Bob",
				LastName:           "Taylor",
				Email:              "bob.taylor@northgate.test",
				Phone:              "+44 7700 900201",
				Address:            "22 Minster Avenue, York",
				EmergencyContact:   "Sarah Taylor, +44 7700 900202",
				Department:         "Stockroom",
				JobTitle:           "Stockroom Assistant",
				EmploymentStatus:   "active",
				SalaryBand:         "A",
				AccessibilityNotes: "Requires advance notice for late rota changes.",
				PrivateHRNotes:     "Recently completed probation period.",
			},
		},
	}

	for _, user := range users {
		if err := createSeedUser(db, user); err != nil {
			return fmt.Errorf("seed user %q: %w", user.Username, err)
		}
	}

	return nil
}

func createSeedUser(db *sql.DB, user seedUser) error {
	exists, err := userExists(db, user.Username)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	result, err := db.Exec(
		`INSERT INTO users (username, password_hash, role)
		 VALUES (?, ?, ?)`,
		user.Username,
		string(passwordHash),
		user.Role,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted user id: %w", err)
	}

	if err := insertEmployeeRecord(db, userID, user.Record); err != nil {
		return fmt.Errorf("insert employee record: %w", err)
	}

	return nil
}

func userExists(db *sql.DB, username string) (bool, error) {
	var count int

	err := db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE username = ?`,
		username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return count > 0, nil
}

func insertEmployeeRecord(db *sql.DB, userID int64, record seedEmployeeRecord) error {
	_, err := db.Exec(
		`INSERT INTO employee_records (
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
			accessibility_notes,
			private_hr_notes,
			last_updated_by
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		record.FirstName,
		record.LastName,
		record.Email,
		record.Phone,
		record.Address,
		record.EmergencyContact,
		record.Department,
		record.JobTitle,
		record.EmploymentStatus,
		record.SalaryBand,
		record.AccessibilityNotes,
		record.PrivateHRNotes,
		userID,
	)
	if err != nil {
		return fmt.Errorf("insert employee record: %w", err)
	}

	return nil
}
