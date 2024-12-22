package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},     // Valid email
		{"user.name@domain.co", true},  // Valid email with subdomain
		{"user@domain", false},         // Missing TLD
		{"@domain.com", false},         // Missing username
		{"username@", false},           // Missing domain
		{"username@domain.", false},    // Invalid TLD
		{"username@domain.c", true},    // Valid single-letter TLD
		{"user+name@domain.com", true}, // Valid email with +
		{"user@domain..com", false},    // Double dot in domain
		{"user@.com", false},           // Dot at start of domain
		{"", false},                    // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := IsEmail(tt.email)
			if result != tt.expected {
				t.Errorf("IsEmail(%q) = %v; want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestGenerateXTid(t *testing.T) {
	nodeName := "testNode"
	xTid := GenerateXTid(nodeName)

	// Check the length of the generated xTid
	if len(xTid) != 22 {
		t.Errorf("Expected xTid length of 22, but got %d", len(xTid))
	}

	// Check if the xTid starts with the expected node name (up to 5 characters)
	expectedPrefix := nodeName[:5]
	if !startsWith(xTid, expectedPrefix) {
		t.Errorf("xTid does not start with expected prefix: got %s, want prefix %s", xTid, expectedPrefix)
	}

	// Check if the xTid includes the current date in "yymmdd" format
	expectedDate := time.Now().Format("060102")
	if !containsDate(xTid, expectedDate) {
		t.Errorf("xTid does not contain the expected date: got %s, want date %s", xTid, expectedDate)
	}
}
func TestIsStructEmpty(t *testing.T) {
	type TestStruct struct {
		Field1 string
		Field2 int
		Field3 bool
	}

	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "Empty struct",
			input:    TestStruct{},
			expected: true,
		},
		{
			name: "Non-empty struct with string",
			input: TestStruct{
				Field1: "value",
			},
			expected: false,
		},
		{
			name: "Non-empty struct with int",
			input: TestStruct{
				Field2: 1,
			},
			expected: false,
		},
		{
			name: "Non-empty struct with bool",
			input: TestStruct{
				Field3: true,
			},
			expected: false,
		},
		{
			name: "Non-empty struct with all fields",
			input: TestStruct{
				Field1: "value",
				Field2: 1,
				Field3: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStructEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsStructEmpty(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
func TestProjectName(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		projectName    string
		expected       string
		goModContent   string
		expectGoModErr bool
	}{
		{
			name:        "Service name environment variable set",
			serviceName: "service-name",
			expected:    "service-name",
		},
		{
			name:        "Project name environment variable set",
			projectName: "project-name",
			expected:    "project-name",
		},
		{
			name:         "No environment variables, valid go.mod",
			goModContent: "module github.com/sing3demons/car-management-system\n",
			expected:     "car-management-system",
		},
		{
			name:           "No environment variables, error reading go.mod",
			expectGoModErr: true,
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Backup and restore environment variables
			restoreEnvVars := backupAndRestoreEnvVars()
			defer restoreEnvVars()

			// Set test environment variables
			os.Setenv("SERVICE_NAME", tt.serviceName)
			os.Setenv("PROJECT_NAME", tt.projectName)

			// Mock go.mod file if needed
			var cleanup func()
			var err error
			if tt.goModContent != "" {
				cleanup, err = mockGoModFile(tt.goModContent)
				if err != nil {
					t.Fatalf("Failed to mock go.mod file: %v", err)
				}
				defer cleanup()
			} else if tt.expectGoModErr {
				cleanup, err = mockGoModError()
				if err != nil {
					t.Fatalf("Failed to mock go.mod error: %v", err)
				}
				defer cleanup()
			}

			// Call ProjectName and validate results
			result := ProjectName()
			if result != tt.expected {
				t.Errorf("ProjectName() = %v; want %v", result, tt.expected)
			}
		})
	}
}
func backupAndRestoreEnvVars() func() {
	originalServiceName := os.Getenv("SERVICE_NAME")
	originalProjectName := os.Getenv("PROJECT_NAME")
	return func() {
		os.Setenv("SERVICE_NAME", originalServiceName)
		os.Setenv("PROJECT_NAME", originalProjectName)
	}
}

func mockGoModFile(content string) (cleanup func(), err error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "gomod-mock")
	if err != nil {
		return nil, err
	}

	// Create the go.mod file in the temporary directory
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
		return nil, err
	}

	// Get the current working directory to restore later
	originalWd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Change to the temporary directory
	if err := os.Chdir(tmpDir); err != nil {
		return nil, err
	}

	// Return a cleanup function to restore the working directory and remove the temp files
	return func() {
		os.Chdir(originalWd)
		os.RemoveAll(tmpDir)
	}, nil
}

func mockGoModError() (cleanup func(), err error) {
	originalWd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "go.mod-mock")
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(tmpDir); err != nil {
		return nil, err
	}

	return func() {
		os.RemoveAll(tmpDir)
		os.Chdir(originalWd)
	}, nil
}
