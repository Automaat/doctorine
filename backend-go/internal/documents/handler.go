package documents

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const maxUploadBytes = 50 << 20

type Handler struct {
	store     *Store
	uploadDir string
	logger    *slog.Logger
}

func NewHandler(store *Store, uploadDir string, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, uploadDir: uploadDir, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list documents", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Upload must be multipart and under 50 MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, "File is required")
		return
	}
	defer file.Close()

	params, detail := h.paramsFromForm(r, header.Filename)
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, detail)
		return
	}
	params.OriginalFilename = safeFilename(header.Filename)
	params.StorageName, err = newStorageName(header.Filename)
	if err != nil {
		h.logger.Error("new storage name", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	params.ContentType = header.Header.Get("content-type")

	path := filepath.Join(h.uploadDir, params.StorageName)
	written, shaHex, detectedType, err := writeUpload(path, file, params.ContentType)
	if err != nil {
		_ = os.Remove(path)
		h.logger.Error("write upload", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	params.SizeBytes = written
	params.SHA256Hex = shaHex
	params.ContentType = detectedType

	item, err := h.store.Create(r.Context(), params)
	if err != nil {
		_ = os.Remove(path)
		h.logger.Error("create document", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Document not found")
			return
		}
		h.logger.Error("get document", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	path := filepath.Join(h.uploadDir, item.StorageName)
	file, err := os.Open(path)
	if err != nil {
		h.logger.Error("open document file", "err", err, "document_id", id)
		httputil.WriteDetailError(w, http.StatusNotFound, "Document file not found")
		return
	}
	defer file.Close()

	w.Header().Set("content-type", item.ContentType)
	w.Header().Set("content-disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": item.OriginalFilename,
	}))
	http.ServeContent(w, r, item.OriginalFilename, time.Time{}, file)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.store.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Document not found")
			return
		}
		h.logger.Error("delete document", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if err := os.Remove(filepath.Join(h.uploadDir, item.StorageName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		h.logger.Warn("remove document file", "err", err, "document_id", id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) paramsFromForm(r *http.Request, filename string) (CreateParams, string) {
	title := healthstatus.CleanString(r.FormValue("title"))
	if title == "" {
		title = healthstatus.CleanString(filename)
	}
	if title == "" {
		return CreateParams{}, "Title is required"
	}
	docType := healthstatus.CleanString(r.FormValue("document_type"))
	if docType == "" {
		docType = "medical"
	}
	issuedAt, err := healthstatus.ParseOptionalDate(healthstatus.NilIfEmpty(r.FormValue("issued_at")), "issued_at")
	if err != nil {
		return CreateParams{}, err.Error()
	}
	illnessID, detail := optionalInt(r.FormValue("illness_id"), "illness_id")
	if detail != "" {
		return CreateParams{}, detail
	}
	examinationID, detail := optionalInt(r.FormValue("examination_id"), "examination_id")
	if detail != "" {
		return CreateParams{}, detail
	}
	return CreateParams{
		Title:         title,
		DocumentType:  docType,
		IssuedAt:      issuedAt,
		Notes:         healthstatus.NilIfEmpty(r.FormValue("notes")),
		IllnessID:     illnessID,
		ExaminationID: examinationID,
	}, ""
}

func writeUpload(path string, src io.Reader, declaredType string) (int64, string, string, error) {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, "", "", err
	}
	defer out.Close()

	head := make([]byte, 512)
	read, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return 0, "", "", err
	}
	head = head[:read]
	contentType := declaredType
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(head)
	}
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(out, hasher), io.MultiReader(bytes.NewReader(head), src))
	if err != nil {
		return 0, "", "", err
	}
	if written == 0 {
		return 0, "", "", errors.New("empty upload")
	}
	return written, hex.EncodeToString(hasher.Sum(nil)), contentType, nil
}

func optionalInt(value string, field string) (*int, string) {
	cleaned := healthstatus.CleanString(value)
	if cleaned == "" {
		return nil, ""
	}
	parsed, err := strconv.Atoi(cleaned)
	if err != nil || parsed <= 0 {
		return nil, field + " must be a positive integer"
	}
	return &parsed, ""
}

func parseID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid id")
		return 0, false
	}
	return id, true
}
