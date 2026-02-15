package scripting

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Input validation limits to prevent abuse
const (
	MaxUsernameLen  = 30
	MaxPasswordLen  = 128
	MaxRealNameLen  = 60
	MaxLocationLen  = 60
	MaxEmailLen     = 128
	MaxSubjectLen   = 128
	MaxMessageLen   = 8192  // 8KB message body
	MaxChatLen      = 512   // Chat messages
	MaxFilenameLen  = 255
	MaxPathLen      = 4096
)

// ValidateInput performs common input validation checks.
type ValidateInput struct{}

// ValidateString checks string length and basic content validation.
func (v *ValidateInput) ValidateString(value, fieldName string, maxLen int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s contains invalid UTF-8", fieldName)
	}
	
	length := utf8.RuneCountInString(value)
	if length > maxLen {
		return fmt.Errorf("%s too long (max %d characters)", fieldName, maxLen)
	}
	
	return nil
}

// ValidateUsername checks username requirements.
func (v *ValidateInput) ValidateUsername(username string) error {
	if err := v.ValidateString(username, "username", MaxUsernameLen); err != nil {
		return err
	}
	
	if len(username) < 2 {
		return fmt.Errorf("username too short (minimum 2 characters)")
	}
	
	// Check for valid username characters (alphanumeric, underscore, hyphen)
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		     (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("username contains invalid characters (use letters, numbers, _ or -)")
		}
	}
	
	return nil
}

// ValidatePassword checks password requirements.
func (v *ValidateInput) ValidatePassword(password string) error {
	if err := v.ValidateString(password, "password", MaxPasswordLen); err != nil {
		return err
	}
	
	if len(password) < 6 {
		return fmt.Errorf("password too short (minimum 6 characters)")
	}
	
	return nil
}

// ValidateEmail checks basic email format.
func (v *ValidateInput) ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}
	
	if err := v.ValidateString(email, "email", MaxEmailLen); err != nil {
		return err
	}
	
	// Basic email validation - check for @ with domain part
	atIndex := strings.Index(email, "@")
	if atIndex <= 0 || atIndex == len(email)-1 {
		return fmt.Errorf("invalid email format")
	}
	
	// Check for at least one dot in domain part
	domain := email[atIndex+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("invalid email format")
	}
	
	return nil
}

// ValidateMessageBody checks message content.
func (v *ValidateInput) ValidateMessageBody(body string) error {
	if err := v.ValidateString(body, "message body", MaxMessageLen); err != nil {
		return err
	}
	
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("message body cannot be empty")
	}
	
	return nil
}

// ValidateChatMessage checks chat message content.
func (v *ValidateInput) ValidateChatMessage(text string) error {
	if err := v.ValidateString(text, "chat message", MaxChatLen); err != nil {
		return err
	}
	
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("chat message cannot be empty")
	}
	
	return nil
}

// ValidateFilename checks filename for path traversal and invalid characters.
func (v *ValidateInput) ValidateFilename(filename string) error {
	if err := v.ValidateString(filename, "filename", MaxFilenameLen); err != nil {
		return err
	}
	
	// Clean the path and verify it equals the base name (no directory components)
	cleaned := filepath.Clean(filename)
	base := filepath.Base(cleaned)
	if cleaned != base || base == "." || base == ".." {
		return fmt.Errorf("filename contains path components")
	}
	
	// Check for path separators (belt and suspenders)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("filename contains path separators")
	}
	
	// Check for null bytes and control characters
	for _, r := range filename {
		if r < 32 || r == 127 {
			return fmt.Errorf("filename contains control characters")
		}
	}
	
	return nil
}

// SanitizeForDisplay removes or escapes control characters for display.
func (v *ValidateInput) SanitizeForDisplay(input string) string {
	var result strings.Builder
	for _, r := range input {
		// Keep printable characters, newlines, and tabs
		if (r >= 32 && r < 127) || r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(r)
		} else if r >= 128 {
			// Keep valid UTF-8 high characters
			result.WriteRune(r)
		}
		// Skip other control characters
	}
	return result.String()
}
