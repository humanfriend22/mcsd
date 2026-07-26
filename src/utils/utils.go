package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

func ValidateID(id string) error {
	if !validName.MatchString(id) {
		return fmt.Errorf("invalid instance ID %q: use only letters, digits, hyphens, underscores (max 63 chars)", id)
	}
	if strings.Contains(id, "..") || strings.Contains(id, "/") {
		return fmt.Errorf("invalid instance ID %q: must not contain path separators", id)
	}
	return nil
}

func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
