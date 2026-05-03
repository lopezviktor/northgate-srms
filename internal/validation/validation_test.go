package validation

import "testing"

func TestIsValidPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UK style number", "+44 7700 900001", true},
		{"valid with hyphens", "+44-7700-900001", true},
		{"too short", "123", false},
		{"letters not allowed", "abcdefg", false},
		{"too long", "+44 7700 900001 99999", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPhone(tt.input)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid email", "alice.morgan@northgate.test", true},
		{"missing at symbol", "not-an-email", false},
		{"missing local part", "@northgate.test", false},
		{"missing domain dot", "alice@northgate", false},
		{"missing final domain part", "alice@northgate.", false},
		{"too short", "a@b", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.input)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid name", "Alice", true},
		{"valid with spaces trimmed", " Alice ", true},
		{"too short", "A", false},
		{"empty", "", false},
		{"too long", "ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXY", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsAllowedDepartment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"sales allowed", "Sales", true},
		{"stockroom allowed", "Stockroom", true},
		{"hr allowed", "HR", true},
		{"invalid department", "Finance", false},
		{"empty", "", false},
		{"lowercase not allowed", "sales", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowedDepartment(tt.input)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsAllowedEmploymentStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"active allowed", "active", true},
		{"on leave allowed", "on_leave", true},
		{"terminated allowed", "terminated", true},
		{"invalid status", "suspended", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowedEmploymentStatus(tt.input)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsAllowedSalaryBand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"A allowed", "A", true},
		{"C allowed", "C", true},
		{"E allowed", "E", true},
		{"invalid band", "F", false},
		{"lowercase not allowed", "a", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowedSalaryBand(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
