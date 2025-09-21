package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/sirupsen/logrus"
)

// FileValidationService handles file validation and security checks
type FileValidationService struct {
	virusScanningEnabled bool
	clamAVHost           string
	clamAVPort           string
}

// NewFileValidationService creates a new file validation service
func NewFileValidationService(cfg *config.Config) *FileValidationService {
	return &FileValidationService{
		virusScanningEnabled: cfg.EnableVirusScanning,
		clamAVHost:           "localhost", // Could be configurable
		clamAVPort:           "3310",      // Default ClamAV port
	}
}

// ValidateFile performs comprehensive file validation
func (s *FileValidationService) ValidateFile(ctx context.Context, filename, declaredMimeType string, content io.Reader) (*models.FileValidationResult, error) {
	result := &models.FileValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Read content into buffer for multiple validations
	buf := &bytes.Buffer{}
	size, err := io.Copy(buf, content)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read file content: %v", err))
		return result, nil
	}

	contentBytes := buf.Bytes()
	result.FileSize = size

	// Calculate content hash
	hash := sha256.Sum256(contentBytes)
	result.ContentHash = hex.EncodeToString(hash[:])

	// Validate file size
	if err := s.validateFileSize(size, result); err != nil {
		return result, err
	}

	// Validate filename
	if err := s.validateFilename(filename, result); err != nil {
		return result, err
	}

	// Detect MIME type
	detectedMimeType := s.detectMimeType(filename, contentBytes)
	result.DetectedMimeType = detectedMimeType

	// Validate MIME type consistency
	s.validateMimeTypeConsistency(declaredMimeType, detectedMimeType, result)

	// Check for malicious content
	s.checkMaliciousContent(filename, contentBytes, result)

	// Check file extension security
	s.validateFileExtension(filename, result)

	// Validate content against known file signatures
	s.validateFileSignature(contentBytes, detectedMimeType, result)

	return result, nil
}

// CalculateContentHash calculates SHA-256 hash from reader
func (s *FileValidationService) CalculateContentHash(content io.Reader) (string, int64, error) {
	hash := sha256.New()
	size, err := io.Copy(hash, content)
	if err != nil {
		return "", 0, fmt.Errorf("failed to calculate hash: %w", err)
	}

	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, size, nil
}

// validateFileSize checks if file size is within allowed limits
func (s *FileValidationService) validateFileSize(size int64, result *models.FileValidationResult) error {
	if size == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "File is empty")
	}

	if size > models.MaxFileSize {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File size (%d bytes) exceeds maximum allowed size (%d bytes)", size, models.MaxFileSize))
	}

	return nil
}

// validateFilename checks filename for security issues
func (s *FileValidationService) validateFilename(filename string, result *models.FileValidationResult) error {
	if strings.TrimSpace(filename) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Filename cannot be empty")
		return nil
	}

	// Check for dangerous characters
	dangerousChars := []string{"../", "..\\", "<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Filename contains dangerous character: %s", char))
		}
	}

	// Check filename length
	if len(filename) > 255 {
		result.IsValid = false
		result.Errors = append(result.Errors, "Filename is too long (max 255 characters)")
	}

	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		result.IsValid = false
		result.Errors = append(result.Errors, "Filename contains null bytes")
	}

	return nil
}

// validateFileExtension checks for dangerous file extensions
func (s *FileValidationService) validateFileExtension(filename string, result *models.FileValidationResult) {
	ext := strings.ToLower(filepath.Ext(filename))

	// List of dangerous executable extensions
	dangerousExts := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".vbe",
		".js", ".jar", ".msi", ".dll", ".deb", ".rpm", ".dmg", ".pkg",
		".sh", ".bash", ".zsh", ".fish", ".csh", ".ksh", ".ps1", ".psm1",
		".py", ".rb", ".pl", ".php", ".asp", ".aspx", ".jsp", ".war",
		".ipa", ".apk", ".app", ".gadget", ".workflow",
	}

	for _, dangerousExt := range dangerousExts {
		if ext == dangerousExt {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("File type '%s' is not allowed for security reasons", ext))
			return
		}
	}

	// List of potentially suspicious extensions (warnings only)
	suspiciousExts := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2"}
	for _, suspiciousExt := range suspiciousExts {
		if ext == suspiciousExt {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Archive file detected (%s). Please scan contents before extraction", ext))
			break
		}
	}
}

// detectMimeType detects MIME type using file content and extension
func (s *FileValidationService) detectMimeType(filename string, content []byte) string {
	// First, try to detect by content (magic bytes)
	if detectedType := s.detectMimeTypeByContent(content); detectedType != "" {
		return detectedType
	}

	// Fall back to extension-based detection
	ext := filepath.Ext(filename)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	return "application/octet-stream"
}

// detectMimeTypeByContent detects MIME type by examining file headers (magic bytes)
func (s *FileValidationService) detectMimeTypeByContent(content []byte) string {
	if len(content) == 0 {
		return ""
	}

	// Common file signatures
	signatures := map[string]string{
		// Images
		"\xFF\xD8\xFF":      "image/jpeg",
		"\x89PNG\r\n\x1A\n": "image/png",
		"GIF87a":            "image/gif",
		"GIF89a":            "image/gif",
		"RIFF":              "image/webp", // Will need additional check
		"\x00\x00\x01\x00":  "image/x-icon",

		// Documents
		"PK\x03\x04":                       "application/zip", // Also used by DOCX, XLSX, etc.
		"%PDF-":                            "application/pdf",
		"\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1": "application/msword", // DOC, XLS, PPT

		// Archives
		"Rar!\x1A\x07\x00":   "application/x-rar-compressed",
		"\x1F\x8B":           "application/gzip",
		"7z\xBC\xAF\x27\x1C": "application/x-7z-compressed",

		// Audio/Video
		"ID3":              "audio/mpeg",
		"\xFF\xFB":         "audio/mpeg",
		"OggS":             "audio/ogg",
		"\x1A\x45\xDF\xA3": "video/webm",

		// Text
		"\xEF\xBB\xBF": "text/plain", // UTF-8 BOM
	}

	for signature, mimeType := range signatures {
		if len(content) >= len(signature) && string(content[:len(signature)]) == signature {
			// Special case for RIFF files (need to check for WEBP)
			if signature == "RIFF" && len(content) >= 12 {
				if string(content[8:12]) == "WEBP" {
					return "image/webp"
				}
				return "audio/wav" // Default for RIFF
			}
			return mimeType
		}
	}

	// Check for XML (look for opening XML tag)
	if len(content) > 5 && (bytes.HasPrefix(content, []byte("<?xml")) || bytes.HasPrefix(content, []byte("<html")) || bytes.HasPrefix(content, []byte("<!DOCTYPE"))) {
		if bytes.Contains(content[:100], []byte("html")) {
			return "text/html"
		}
		return "application/xml"
	}

	// Check for text files (basic heuristic)
	if s.isTextContent(content) {
		return "text/plain"
	}

	return ""
}

// isTextContent checks if content appears to be text
func (s *FileValidationService) isTextContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 512 bytes for non-printable characters
	checkSize := 512
	if len(content) < checkSize {
		checkSize = len(content)
	}

	nonPrintable := 0
	for i := 0; i < checkSize; i++ {
		b := content[i]
		// Allow common whitespace characters
		if b == '\t' || b == '\n' || b == '\r' || (b >= 32 && b <= 126) {
			continue
		}
		// Allow UTF-8 characters
		if b >= 128 {
			continue
		}
		nonPrintable++
	}

	// If more than 30% non-printable, probably binary
	return float64(nonPrintable)/float64(checkSize) < 0.3
}

// validateMimeTypeConsistency checks if declared and detected MIME types match
func (s *FileValidationService) validateMimeTypeConsistency(declared, detected string, result *models.FileValidationResult) {
	if declared == "" {
		return
	}

	if detected != declared {
		// Some acceptable mismatches
		acceptableMismatches := map[string][]string{
			"application/octet-stream": {"*"}, // Generic type is acceptable
			"text/plain":               {"application/octet-stream"},
			"application/zip":          {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		}

		if accepted, exists := acceptableMismatches[detected]; exists {
			for _, acceptedType := range accepted {
				if acceptedType == "*" || acceptedType == declared {
					return
				}
			}
		}

		result.Warnings = append(result.Warnings, fmt.Sprintf("Declared MIME type (%s) differs from detected type (%s)", declared, detected))
	}
}

// checkMaliciousContent performs additional security checks
func (s *FileValidationService) checkMaliciousContent(filename string, content []byte, result *models.FileValidationResult) {
	// Check for suspicious patterns in filenames
	suspiciousPatterns := []string{
		"autorun.inf",
		"desktop.ini",
		".htaccess",
		"web.config",
		"config.php",
		"wp-config.php",
		".env",
		"id_rsa",
		"id_dsa",
		"private.key",
	}

	lowerFilename := strings.ToLower(filename)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerFilename, pattern) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Suspicious filename pattern detected: %s", pattern))
		}
	}

	// Check for embedded executables in content
	if len(content) > 2 {
		// Check for PE header (Windows executables)
		if bytes.Contains(content, []byte("MZ")) && bytes.Contains(content, []byte("PE\x00\x00")) {
			result.IsValid = false
			result.Errors = append(result.Errors, "File contains embedded Windows executable")
		}

		// Check for ELF header (Linux executables)
		if len(content) >= 4 && string(content[:4]) == "\x7FELF" {
			result.IsValid = false
			result.Errors = append(result.Errors, "File contains embedded Linux executable")
		}

		// Check for Mach-O header (macOS executables)
		if len(content) >= 4 && (string(content[:4]) == "\xFE\xED\xFA\xCE" || string(content[:4]) == "\xFE\xED\xFA\xCF") {
			result.IsValid = false
			result.Errors = append(result.Errors, "File contains embedded macOS executable")
		}
	}

	// Check for script injections in text files
	if s.isTextContent(content) {
		suspiciousScripts := []string{
			"<script",
			"javascript:",
			"vbscript:",
			"onload=",
			"onerror=",
			"eval(",
			"exec(",
			"system(",
			"shell_exec(",
		}

		contentStr := strings.ToLower(string(content))
		for _, script := range suspiciousScripts {
			if strings.Contains(contentStr, script) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Potentially suspicious script content detected: %s", script))
			}
		}
	}
}

// validateFileSignature validates that file content matches its declared type
func (s *FileValidationService) validateFileSignature(content []byte, detectedMimeType string, result *models.FileValidationResult) {
	if len(content) < 4 {
		return
	}

	// Additional validation for specific file types
	switch detectedMimeType {
	case "image/jpeg":
		if !bytes.HasPrefix(content, []byte("\xFF\xD8\xFF")) {
			result.Warnings = append(result.Warnings, "JPEG file signature validation failed")
		}
	case "image/png":
		if !bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1A\n")) {
			result.Warnings = append(result.Warnings, "PNG file signature validation failed")
		}
	case "application/pdf":
		if !bytes.HasPrefix(content, []byte("%PDF-")) {
			result.Warnings = append(result.Warnings, "PDF file signature validation failed")
		}
	}
}

// Enhanced Security Validation Methods

// ValidateFileWithVirusScan performs comprehensive validation including virus scanning
func (s *FileValidationService) ValidateFileWithVirusScan(ctx context.Context, filename, declaredMimeType string, content io.Reader) (*models.FileValidationResult, error) {
	// First perform standard validation
	result, err := s.ValidateFile(ctx, filename, declaredMimeType, content)
	if err != nil {
		return result, err
	}

	// If virus scanning is enabled and file passed initial validation
	if s.virusScanningEnabled && result.IsValid {
		// Reset reader position
		if seeker, ok := content.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		}

		virusScanResult, err := s.scanForViruses(ctx, content)
		if err != nil {
			logrus.Warnf("Virus scan failed for file %s: %v", filename, err)
			result.Warnings = append(result.Warnings, "Virus scan could not be completed")
		} else {
			result.VirusScanResult = virusScanResult
			if !virusScanResult.Clean {
				result.IsValid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Virus detected: %s", virusScanResult.ThreatName))
			}
		}
	}

	return result, nil
}

// scanForViruses performs virus scanning using ClamAV
func (s *FileValidationService) scanForViruses(ctx context.Context, content io.Reader) (*models.VirusScanResult, error) {
	scanResult := &models.VirusScanResult{
		Clean:     true,
		Engine:    "ClamAV",
		ScannedAt: time.Now(),
	}

	if !s.virusScanningEnabled {
		scanResult.Status = "disabled"
		return scanResult, nil
	}

	// Try to connect to ClamAV daemon
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", s.clamAVHost, s.clamAVPort), 10*time.Second)
	if err != nil {
		scanResult.Status = "error"
		scanResult.Error = fmt.Sprintf("Failed to connect to ClamAV: %v", err)
		return scanResult, fmt.Errorf("ClamAV connection failed: %w", err)
	}
	defer conn.Close()

	// Send INSTREAM command
	_, err = conn.Write([]byte("nINSTREAM\n"))
	if err != nil {
		scanResult.Status = "error"
		scanResult.Error = "Failed to send scan command"
		return scanResult, err
	}

	// Stream file content to ClamAV
	buffer := make([]byte, 8192)
	for {
		n, err := content.Read(buffer)
		if n > 0 {
			// Send chunk size and data
			chunkSize := fmt.Sprintf("%08x", n)
			conn.Write([]byte(chunkSize))
			conn.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			scanResult.Status = "error"
			scanResult.Error = "Failed to stream file content"
			return scanResult, err
		}
	}

	// Send end-of-stream marker
	conn.Write([]byte("00000000"))

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		scanResult.Status = "error"
		scanResult.Error = "Failed to read scan result"
		return scanResult, err
	}

	responseStr := string(response[:n])
	logrus.Debugf("ClamAV response: %s", responseStr)

	// Parse ClamAV response
	if strings.Contains(responseStr, "FOUND") {
		scanResult.Clean = false
		scanResult.Status = "infected"
		// Extract threat name
		parts := strings.Split(responseStr, ": ")
		if len(parts) >= 2 {
			threatParts := strings.Split(parts[1], " ")
			if len(threatParts) > 0 {
				scanResult.ThreatName = threatParts[0]
			}
		}
	} else if strings.Contains(responseStr, "OK") {
		scanResult.Status = "clean"
	} else {
		scanResult.Status = "error"
		scanResult.Error = fmt.Sprintf("Unexpected ClamAV response: %s", responseStr)
	}

	return scanResult, nil
}
