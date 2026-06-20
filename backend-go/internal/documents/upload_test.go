package documents

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestValidateDocumentType(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		detectedType string
		wantErr      error
	}{
		{"pdf", "report.pdf", "application/pdf", nil},
		{"jpg", "photo.jpg", "image/jpeg", nil},
		{"jpeg", "photo.jpeg", "image/jpeg", nil},
		{"png uppercase ext", "scan.PNG", "image/png", nil},
		{"webp", "image.webp", "image/webp", nil},
		{"match with charset suffix", "report.pdf", "application/pdf; charset=binary", nil},
		{"spoofed pdf extension", "evil.pdf", "image/png", errUnsupportedType},
		{"unknown extension", "malware.exe", "application/octet-stream", errUnsupportedType},
		{"no extension", "noext", "application/pdf", errUnsupportedType},
		{"text disguised as pdf", "notes.pdf", "text/plain; charset=utf-8", errUnsupportedType},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDocumentType(tc.filename, tc.detectedType)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("validateDocumentType(%q, %q) = %v, want %v",
					tc.filename, tc.detectedType, err, tc.wantErr)
			}
		})
	}
}

var (
	pdfMagic  = []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n")
	pngMagic  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")
	jpegMagic = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")
	webpMagic = []byte("RIFF\x24\x00\x00\x00WEBPVP8 \x00\x00\x00\x00")
)

func TestWriteUploadAcceptsAllowedTypes(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		content  []byte
		wantType string
	}{
		{"pdf", "lab.pdf", pdfMagic, "application/pdf"},
		{"png", "scan.png", pngMagic, "image/png"},
		{"jpeg", "photo.jpg", jpegMagic, "image/jpeg"},
		{"webp", "image.webp", webpMagic, "image/webp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			written, sha, contentType, err := writeUpload(dir, tc.name, bytes.NewReader(tc.content), tc.filename)
			if err != nil {
				t.Fatalf("writeUpload: unexpected error: %v", err)
			}
			if contentType != tc.wantType {
				t.Fatalf("content type = %q, want %q", contentType, tc.wantType)
			}
			if written != int64(len(tc.content)) {
				t.Fatalf("written = %d, want %d", written, len(tc.content))
			}
			if len(sha) != 64 {
				t.Fatalf("sha256 hex = %q, want 64 chars", sha)
			}
		})
	}
}

func TestWriteUploadRejectsBadUploads(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		content  []byte
		wantErr  error
	}{
		{"spoofed mime", "evil.pdf", pngMagic, errUnsupportedType},
		{"unknown extension", "data.bin", pdfMagic, errUnsupportedType},
		{"empty file", "empty.pdf", nil, errEmptyUpload},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			_, _, _, err := writeUpload(dir, tc.name, bytes.NewReader(tc.content), tc.filename)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("writeUpload err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestUploadReadResponseMapsTypeErrors(t *testing.T) {
	status, detail := uploadReadResponse(errUnsupportedType)
	if status != 422 {
		t.Fatalf("status = %d, want 422", status)
	}
	if !strings.Contains(detail, allowedDocumentTypesLabel) {
		t.Fatalf("detail %q missing allowed types label", detail)
	}
	if status, _ := uploadReadResponse(errEmptyUpload); status != 422 {
		t.Fatalf("empty upload status = %d, want 422", status)
	}
}
