package documents

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// errEmptyUpload and errUnsupportedType are returned while reading an upload so
// the handler can map them to a 422 response.
var (
	errEmptyUpload     = errors.New("empty upload")
	errUnsupportedType = errors.New("unsupported document type")
)

// allowedDocumentTypes maps an allowed lowercase extension to the content types
// that http.DetectContentType must report for that extension. Extension and
// sniffed type have to agree, so a file's declared name cannot disguise its
// real format.
var allowedDocumentTypes = map[string][]string{
	".pdf":  {"application/pdf"},
	".jpg":  {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".png":  {"image/png"},
	".webp": {"image/webp"},
}

// allowedDocumentTypesLabel lists the user-facing accepted formats.
const allowedDocumentTypesLabel = "PDF, JPEG, PNG, WebP"

// validateDocumentType reports whether filename's extension is allowed and its
// sniffed content type agrees with that extension.
func validateDocumentType(filename string, detectedType string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed, ok := allowedDocumentTypes[ext]
	if !ok {
		return errUnsupportedType
	}
	mediaType := detectedType
	if i := strings.IndexByte(mediaType, ';'); i >= 0 {
		mediaType = strings.TrimSpace(mediaType[:i])
	}
	if slices.Contains(allowed, mediaType) {
		return nil
	}
	return errUnsupportedType
}

func safeFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" {
		return "document"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('_')
		}
	}
	cleaned := strings.Trim(b.String(), "._-")
	if cleaned == "" {
		return "document"
	}
	runes := []rune(cleaned)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return cleaned
}

func newStorageName(original string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]) + "-" + safeFilename(original), nil
}
