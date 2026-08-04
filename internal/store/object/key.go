package object

import (
	"path"
	"strings"
)

// ValidateKey checks whether a key is valid and free of path traversal sequences.
func ValidateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if strings.Contains(key, "\x00") {
		return ErrPathTraversal
	}
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return ErrPathTraversal
	}
	if strings.Contains(key, "\\") {
		return ErrPathTraversal
	}

	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == ".." || part == "." {
			return ErrPathTraversal
		}
	}

	cleaned := path.Clean("/" + key)
	if cleaned == "/" || cleaned == "." || strings.HasPrefix(cleaned, "/..") {
		return ErrPathTraversal
	}

	return nil
}

// ValidatePrefixComponent verifies that a tenant or test-run prefix segment is valid.
func ValidatePrefixComponent(comp string) error {
	if comp == "" {
		return nil
	}
	if strings.Contains(comp, "\x00") || strings.Contains(comp, "/") || strings.Contains(comp, "\\") || comp == ".." || comp == "." {
		return ErrPathTraversal
	}
	return nil
}

// BuildKey constructs a normalized key incorporating tenantID, testRunID, and baseKey.
func BuildKey(tenantID, testRunID, baseKey string) (string, error) {
	if err := ValidatePrefixComponent(tenantID); err != nil {
		return "", err
	}
	if err := ValidatePrefixComponent(testRunID); err != nil {
		return "", err
	}

	var parts []string
	if tenantID != "" {
		parts = append(parts, tenantID)
	}
	if testRunID != "" {
		parts = append(parts, testRunID)
	}
	if baseKey != "" {
		if err := ValidateKey(baseKey); err != nil {
			return "", err
		}
		parts = append(parts, baseKey)
	}

	if len(parts) == 0 {
		return "", ErrInvalidKey
	}

	fullKey := strings.Join(parts, "/")
	if err := ValidateKey(fullKey); err != nil {
		return "", err
	}
	return fullKey, nil
}
