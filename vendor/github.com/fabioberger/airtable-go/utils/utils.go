package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// SwitchCaseError creates and returns a switch case error.
func SwitchCaseError(name string, value interface{}) error {
	return fmt.Errorf("Unrecognized %s: %v", name, value)
}

// IsValidAPIKey checks for a correctly formatted API key
func IsValidAPIKey(apiKey string) bool {
	return isValidAirtableID(apiKey, "key")
}

// IsValidBaseID checks for a correctly formatted base ID
func IsValidBaseID(baseID string) bool {
	return isValidAirtableID(baseID, "app")
}

// CheckForValidRecordID checks if a correctly formatted record ID was supplied, returns error if not
func CheckForValidRecordID(recordID string) error {
	if !isValidAirtableID(recordID, "rec") {
		return fmt.Errorf("Invalid recordID encountered: %s", recordID)
	}
	return nil
}

func isValidAirtableID(id, expectedPrefix string) bool {
	regex := regexp.MustCompile("[a-zA-Z0-9]{17}")
	return len(id) == 17 && strings.HasPrefix(id, expectedPrefix) && regex.Match([]byte(id))
}
