package documents

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unicode"
)

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
