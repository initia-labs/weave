package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Utility function to clean the string by trimming spaces and removing ^M characters
func cleanString(input string) string {
	return strings.TrimSpace(strings.ReplaceAll(input, "\r", ""))
}

// UpdateTomlValue updates a TOML file based on the provided key and value.
// The key can be a field in a section (e.g., "api.enable") or a top-level field (e.g., "minimum-gas-prices").
func UpdateTomlValue(filePath, key, value string) error {
	value = cleanString(value)
	// Open the TOML file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Determine if the key has a section (e.g., "api.enable") or is a top-level field (e.g., "minimum-gas-prices")
	var section, field string
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 2 {
		section = parts[0] // e.g., "api"
		field = parts[1]   // e.g., "enable"
	} else {
		field = key // e.g., "minimum-gas-prices"
	}

	// Slice to store updated file lines
	var updatedLines []string
	var currentSection string
	inTargetSection := false

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check if the line is a section header (e.g., [api])
		if isSectionHeader(trimmedLine) {
			currentSection = getSectionName(trimmedLine)
			inTargetSection = (currentSection == section)
		}

		// Modify the field if it's in the correct section or at the top-level
		if shouldModifyField(inTargetSection, currentSection, field, trimmedLine) {
			line = fmt.Sprintf(`%s = "%s"`, field, value)
		}

		// Add the line to the updated content
		updatedLines = append(updatedLines, line)
	}

	// Check for any scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Write the modified lines back to the file
	err = os.WriteFile(filePath, []byte(strings.Join(updatedLines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

// isSectionHeader checks if a line is a section header (e.g., [api]).
func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

// getSectionName extracts the section name from a section header (e.g., [api] -> api).
func getSectionName(header string) string {
	return strings.Trim(header, "[]")
}

// shouldModifyField checks if the current line should be modified.
func shouldModifyField(inTargetSection bool, currentSection, field, line string) bool {
	// If there is no section (top-level), and the line starts with the field, modify it
	if currentSection == "" && strings.HasPrefix(strings.TrimSpace(line), field) {
		return true
	}

	// If we are in the target section and the line starts with the field, modify it
	if inTargetSection && strings.HasPrefix(strings.TrimSpace(line), field) {
		return true
	}

	return false
}
