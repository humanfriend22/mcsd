package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

func ValidateID(id string) error {
	if !validName.MatchString(id) {
		return fmt.Errorf("invalid instance ID %q: use only letters, digits, hyphens, underscores (max 63 chars)", id)
	}
	if strings.Contains(id, ".") || strings.Contains(id, "/") {
		return fmt.Errorf("invalid instance ID %q: must not contain path separators", id)
	}
	return nil
}

// Joins base and path, but ensures the result is within base. Returns an error if the result would be outside base.
func SafeJoin(base, path string) (string, error) {
	abs := filepath.Join(base, filepath.Clean("/"+path))
	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return "", &ValidationError{Message: "path outside instance directory"}
	}
	return abs, nil
}
