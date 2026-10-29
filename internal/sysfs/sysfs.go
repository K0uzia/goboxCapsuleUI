package sysfs

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const MaxSysfsFileSize = 4096

// ═══════════════════════════════════════════════════════════════════
// SECURITY VALIDATION
// ═══════════════════════════════════════════════════════════════════

// ValidateSysfsName checks that a sysfs filename contains no dangerous characters.
//
// Rejects empty names, path traversal (..) and invalid characters.
// Only allows: a-z A-Z 0-9 - _ : .
func ValidateSysfsName(name string) error {
	if name == "" {
		return fmt.Errorf("empty name")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return fmt.Errorf("path traversal detected: %s", name)
	}

	for _, c := range name {
		valid := (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == ':' || c == '.'

		if !valid {
			return fmt.Errorf("invalid character in name: %s", name)
		}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// FILE READING (simple and optimized versions)
// ═══════════════════════════════════════════════════════════════════

// ReadFile reads a sysfs file and returns its content.
//
// Limits reading to [MaxSysfsFileSize] bytes for security reasons.
// Returns an error if the file doesn't exist or is not readable.
func ReadFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, MaxSysfsFileSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(string(buf[:n])), nil
}

// ReadFileWithBuffer reads a sysfs file using an external buffer.
//
// Optimized for intensive loops where the same buffer can be shared
// across multiple calls. The buffer must have capacity >= [MaxSysfsFileSize].
func ReadFileWithBuffer(path string, buf []byte) (string, error) {
	if len(buf) < MaxSysfsFileSize {
		return "", fmt.Errorf("buffer too small")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(string(buf[:n])), nil
}

// ReadFileOptional reads a sysfs file and returns empty string if missing.
//
// Unlike [ReadFile], doesn't return an error if the file doesn't exist.
// Useful for optional files like /sys/class/net/eth0/wireless.
func ReadFileOptional(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		return "", nil
	}
	return data, err
}

// ═══════════════════════════════════════════════════════════════════
// TYPE CONVERSION (simple versions)
// ═══════════════════════════════════════════════════════════════════

// ReadInt reads a sysfs file and converts its content to integer.
//
// Returns an error if the conversion fails (non-numeric content).
func ReadInt(path string) (int, error) {
	s, err := ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parsing int %s: %w", path, err)
	}
	return val, nil
}

// ReadFloat reads a sysfs file and converts its content to float64.
//
// Returns an error if the conversion fails (non-numeric content).
func ReadFloat(path string) (float64, error) {
	s, err := ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing float %s: %w", path, err)
	}
	return val, nil
}

// ReadBool reads a sysfs file and converts its content to boolean.
//
// Accepts values: "1", "0", "true", "false", "yes", "no" (case-insensitive).
// Returns an error if the value is not recognized.
func ReadBool(path string) (bool, error) {
	s, err := ReadFile(path)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(s) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value: %s", s)
	}
}

// ═══════════════════════════════════════════════════════════════════
// BUFFER-OPTIMIZED VERSIONS (for loops)
// ═══════════════════════════════════════════════════════════════════

// ReadIntWithBuffer reads a sysfs file, converts to integer and reuses an external buffer.
//
// Prefer over [ReadInt] in loops to avoid repeated allocations.
// Use the same buffer for all calls within the loop.
func ReadIntWithBuffer(path string, buf []byte) (int, error) {
	s, err := ReadFileWithBuffer(path, buf)
	if err != nil {
		return 0, err
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parsing int %s: %w", path, err)
	}
	return val, nil
}

// ReadFloatWithBuffer reads a sysfs file, converts to float64 and reuses an external buffer.
//
// Prefer over [ReadFloat] in loops to avoid repeated allocations.
// Use the same buffer for all calls within the loop.
func ReadFloatWithBuffer(path string, buf []byte) (float64, error) {
	s, err := ReadFileWithBuffer(path, buf)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing float %s: %w", path, err)
	}
	return val, nil
}

// ReadBoolWithBuffer reads a sysfs file, converts to boolean and reuses an external buffer.
//
// Prefer over [ReadBool] in loops to avoid repeated allocations.
// Accepts the same values as [ReadBool].
func ReadBoolWithBuffer(path string, buf []byte) (bool, error) {
	s, err := ReadFileWithBuffer(path, buf)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(s) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value: %s", s)
	}
}
