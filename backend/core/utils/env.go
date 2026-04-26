// Package utils provides shareable utility functions.
package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadDotEnv reads a .env file and sets the environment variables.
// This is used for local development so you don't have to manually export variables.
func LoadDotEnv(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Split by the first '=' found
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Printf("✅ Loaded environment from: %s\n", filePath)
	return nil
}

// GetEnv reads an environment variable by name.
// If the variable is not set or is empty, it returns the provided fallback value.
func GetEnv(key, fallbackValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallbackValue
	}
	return value
}
