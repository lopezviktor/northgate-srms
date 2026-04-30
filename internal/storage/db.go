package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func CreateSchema(db *sql.DB) error {
	queries := []string{
		createUsersTable,
		createEmployeeRecordsTable,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("execute schema query: %w", err)
		}
	}

	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL CHECK (role IN ('employee', 'admin')),
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

const createEmployeeRecordsTable = `
CREATE TABLE IF NOT EXISTS employee_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL UNIQUE,

	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	phone TEXT NOT NULL,
	address TEXT NOT NULL,
	emergency_contact TEXT NOT NULL,

	department TEXT NOT NULL,
	job_title TEXT NOT NULL,
	employment_status TEXT NOT NULL CHECK (
		employment_status IN ('active', 'on_leave', 'terminated')
	),
	salary_band TEXT NOT NULL CHECK (
		salary_band IN ('A', 'B', 'C', 'D', 'E')
	),

	accessibility_notes TEXT,
	private_hr_notes TEXT,

	last_updated_by INTEGER NOT NULL,
	last_updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,

	FOREIGN KEY (user_id) REFERENCES users(id),
	FOREIGN KEY (last_updated_by) REFERENCES users(id)
);
`
