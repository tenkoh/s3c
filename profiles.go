package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProfileReader reads AWS profiles from ~/.aws/credentials
type ProfileReader struct{}

// NewProfileReader creates a new ProfileReader
func NewProfileReader() *ProfileReader {
	return &ProfileReader{}
}

// GetProfiles returns a list of available AWS profiles
func (pr *ProfileReader) GetProfiles() ([]string, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Path to AWS credentials file
	credentialsPath := filepath.Join(homeDir, ".aws", "credentials")

	// Check if credentials file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if file doesn't exist
	}

	// Open and read credentials file
	file, err := os.Open(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

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
