package validation

import (
	"regexp"
	"strings"
)

var (
	phonePattern    = regexp.MustCompile(`^[0-9+\-\s]+$`)
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

func IsValidUsername(username string) bool {
	username = strings.TrimSpace(username)

	if len(username) < 3 || len(username) > 30 {
		return false
	}

	return usernamePattern.MatchString(username)
}

func IsValidName(value string) bool {
	value = strings.TrimSpace(value)

	return len(value) >= 2 && len(value) <= 50
}

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)

	if len(email) < 5 || len(email) > 120 {
		return false
	}

	atIndex := strings.Index(email, "@")
	dotIndex := strings.LastIndex(email, ".")
	return atIndex > 0 && dotIndex > atIndex+1 && dotIndex < len(email)-1
}

func IsValidPhone(phone string) bool {
	phone = strings.TrimSpace(phone)

	if len(phone) < 7 || len(phone) > 20 {
		return false
	}

	return phonePattern.MatchString(phone)
}

func IsValidAddress(address string) bool {
	address = strings.TrimSpace(address)
	return len(address) >= 5 && len(address) <= 150
}

func IsValidEmergencyContact(value string) bool {
	value = strings.TrimSpace(value)

	return len(value) >= 5 && len(value) <= 120
}

func IsValidJobTitle(value string) bool {
	value = strings.TrimSpace(value)

	return len(value) >= 2 && len(value) <= 60
}

func IsValidAccessibilityNotes(value string) bool {
	return len(strings.TrimSpace(value)) <= 500
}

func IsValidPrivateHRNotes(value string) bool {
	return len(strings.TrimSpace(value)) <= 1000
}

func IsAllowedDepartment(department string) bool {
	switch department {
	case "Sales", "Stockroom", "Management", "HR", "Operations":
		return true
	default:
		return false
	}
}

func IsAllowedEmploymentStatus(status string) bool {
	switch status {
	case "active", "on_leave", "terminated":
		return true
	default:
		return false
	}
}

func IsAllowedSalaryBand(band string) bool {
	switch band {
	case "A", "B", "C", "D", "E":
		return true
	default:
		return false
	}
}
