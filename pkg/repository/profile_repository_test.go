package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileSystemProfileRepository_GetProfiles(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		expectedResult []string
		expectError    bool
	}{
		{
			name: "valid credentials file with multiple profiles",
			fileContent: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[personal]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY

[work]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			expectedResult: []string{"default", "personal", "work"},
			expectError:    false,
		},
		{
			name: "credentials file with comments",
			fileContent: `# This is a comment
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
; This is also a comment
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE`,
			expectedResult: []string{"default", "production"},
			expectError:    false,
		},
		{
			name:           "empty credentials file",
			fileContent:    "",
			expectedResult: nil,
			expectError:    false,
		},
		{
			name: "credentials file with only comments",
			fileContent: `# Just comments
; More comments`,
			expectedResult: nil,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			credentialsPath := filepath.Join(tmpDir, "credentials")
			
			err := os.WriteFile(credentialsPath, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create repository with test path
			repo := NewFileSystemProfileRepositoryWithPath(credentialsPath)

			// Test GetProfiles
			result, err := repo.GetProfiles()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("GetProfiles() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFileSystemProfileRepository_GetProfiles_NonExistentFile(t *testing.T) {
	// Test with non-existent file
	repo := NewFileSystemProfileRepositoryWithPath("/non/existent/path")
	
	result, err := repo.GetProfiles()
	
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice for non-existent file, got: %v", result)
	}
}