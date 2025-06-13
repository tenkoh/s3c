package repository

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemProfileRepository implements ProfileProvider using filesystem
type FileSystemProfileRepository struct {
	credentialsPath string
}

// NewFileSystemProfileRepository creates a new filesystem-based profile repository
func NewFileSystemProfileRepository() *FileSystemProfileRepository {
	homeDir, _ := os.UserHomeDir()
	credentialsPath := filepath.Join(homeDir, ".aws", "credentials")
	
	return &FileSystemProfileRepository{
		credentialsPath: credentialsPath,
	}
}

// NewFileSystemProfileRepositoryWithPath creates a repository with custom path (for testing)
func NewFileSystemProfileRepositoryWithPath(path string) *FileSystemProfileRepository {
	return &FileSystemProfileRepository{
		credentialsPath: path,
	}
}

// GetProfiles returns a list of available AWS profiles
func (r *FileSystemProfileRepository) GetProfiles() ([]string, error) {
	// Check if credentials file exists
	if _, err := os.Stat(r.credentialsPath); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if file doesn't exist
	}

	// Open and read credentials file
	file, err := os.Open(r.credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	return r.parseProfiles(file)
}

// parseProfiles extracts profile names from credentials file content
func (r *FileSystemProfileRepository) parseProfiles(file *os.File) ([]string, error) {
	var profiles []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check if line is a profile section header [profile_name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profileName := strings.Trim(line, "[]")
			profiles = append(profiles, profileName)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	return profiles, nil
}