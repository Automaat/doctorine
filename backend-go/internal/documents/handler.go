package documents

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const (
	maxUploadBytes    = 50 << 20
	maxFormFieldBytes = 1 << 20
)

type uploadedFile struct {
	originalFilename string
	storageName      string
	contentType      string
	sizeBytes        int64
	sha256Hex        string
}

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
	fields, upload, status, detail, err := h.readMultipartUpload(r)
	if detail != "" {
		if upload.storageName != "" {
			_ = removeUpload(h.uploadDir, upload.storageName)
		}
		if err != nil && status >= http.StatusInternalServerError {
			h.logger.Error("read upload", "err", err)
		}
		httputil.WriteDetailError(w, status, detail)
		return
	}

	params, detail := h.paramsFromValues(fields, upload.originalFilename)
	if detail != "" {
		_ = removeUpload(h.uploadDir, upload.storageName)
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, detail)
		return
	}
	params.OriginalFilename = safeFilename(upload.originalFilename)
	params.StorageName = upload.storageName
	params.ContentType = upload.contentType
	params.SizeBytes = upload.sizeBytes
	params.SHA256Hex = upload.sha256Hex

	item, err := h.store.Create(r.Context(), params)
	if err != nil {
		_ = removeUpload(h.uploadDir, params.StorageName)
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
	file, err := openUpload(h.uploadDir, item.StorageName)
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
	if err := removeUpload(h.uploadDir, item.StorageName); err != nil && !errors.Is(err, os.ErrNotExist) {
		h.logger.Warn("remove document file", "err", err, "document_id", id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) readMultipartUpload(
	r *http.Request,
) (map[string]string, uploadedFile, int, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, uploadedFile{}, http.StatusBadRequest, "Upload must be multipart and under 50 MB", err
	}

	fields := map[string]string{}
	var upload uploadedFile
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			status, detail := uploadReadResponse(err)
			if status == 0 {
				status = http.StatusBadRequest
				detail = "Upload must be multipart and under 50 MB"
			}
			return fields, upload, status, detail, err
		}

		name := part.FormName()
		if name == "" {
			continue
		}
		if name == "file" {
			if upload.storageName != "" {
				return fields, upload, http.StatusUnprocessableEntity, "Only one file is allowed", nil
			}
			fileUpload, status, detail, err := h.writeMultipartFile(part)
			upload = fileUpload
			if detail != "" {
				return fields, upload, status, detail, err
			}
			continue
		}
		if _, exists := fields[name]; exists {
			continue
		}
		value, status, detail, err := readMultipartField(part)
		if detail != "" {
			return fields, upload, status, detail, err
		}
		fields[name] = value
	}
	if upload.storageName == "" {
		return fields, upload, http.StatusUnprocessableEntity, "File is required", nil
	}
	return fields, upload, 0, "", nil
}

func (h *Handler) writeMultipartFile(part *multipart.Part) (uploadedFile, int, string, error) {
	filename := part.FileName()
	if healthstatus.CleanString(filename) == "" {
		return uploadedFile{}, http.StatusUnprocessableEntity, "File is required", nil
	}
	storageName, err := newStorageName(filename)
	if err != nil {
		return uploadedFile{}, http.StatusInternalServerError, "Internal Server Error", err
	}
	written, shaHex, detectedType, err := writeUpload(
		h.uploadDir,
		storageName,
		part,
		part.Header.Get("content-type"),
	)
	if err != nil {
		_ = removeUpload(h.uploadDir, storageName)
		status, detail := uploadReadResponse(err)
		if status == 0 {
			status = http.StatusInternalServerError
			detail = "Internal Server Error"
		}
		return uploadedFile{storageName: storageName}, status, detail, err
	}
	return uploadedFile{
		originalFilename: filename,
		storageName:      storageName,
		contentType:      detectedType,
		sizeBytes:        written,
		sha256Hex:        shaHex,
	}, 0, "", nil
}

func readMultipartField(src io.Reader) (string, int, string, error) {
	value, err := io.ReadAll(io.LimitReader(src, maxFormFieldBytes+1))
	if err != nil {
		status, detail := uploadReadResponse(err)
		if status == 0 {
			status = http.StatusBadRequest
			detail = "Upload must be multipart and under 50 MB"
		}
		return "", status, detail, err
	}
	if int64(len(value)) > maxFormFieldBytes {
		return "", http.StatusBadRequest, "Upload form field is too large", nil
	}
	return string(value), 0, "", nil
}

func uploadReadResponse(err error) (int, string) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return http.StatusBadRequest, "Upload must be multipart and under 50 MB"
	}
	return 0, ""
}

func (h *Handler) paramsFromValues(values map[string]string, filename string) (CreateParams, string) {
	title := healthstatus.CleanString(values["title"])
	if title == "" {
		title = healthstatus.CleanString(filename)
	}
	if title == "" {
		return CreateParams{}, "Title is required"
	}
	docType := healthstatus.CleanString(values["document_type"])
	if docType == "" {
		docType = "medical"
	}
	issuedDate, hasIssuedDate, err := healthstatus.ParseOptionalDate(
		healthstatus.NilIfEmpty(values["issued_at"]),
		"issued_at",
	)
	if err != nil {
		return CreateParams{}, err.Error()
	}
	var issuedAt *time.Time
	if hasIssuedDate {
		issuedAt = &issuedDate
	}
	illnessID, detail := optionalInt(values["illness_id"], "illness_id")
	if detail != "" {
		return CreateParams{}, detail
	}
	examinationID, detail := optionalInt(values["examination_id"], "examination_id")
	if detail != "" {
		return CreateParams{}, detail
	}
	return CreateParams{
		Title:         title,
		DocumentType:  docType,
		IssuedAt:      issuedAt,
		Notes:         healthstatus.NilIfEmpty(values["notes"]),
		IllnessID:     illnessID,
		ExaminationID: examinationID,
	}, ""
}

func openUpload(uploadDir string, storageName string) (*os.File, error) {
	root, err := os.OpenRoot(uploadDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(storageName)
}

func removeUpload(uploadDir string, storageName string) error {
	root, err := os.OpenRoot(uploadDir)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(storageName)
}

func writeUpload(
	uploadDir string,
	storageName string,
	src io.Reader,
	declaredType string,
) (int64, string, string, error) {
	root, err := os.OpenRoot(uploadDir)
	if err != nil {
		return 0, "", "", err
	}
	defer root.Close()

	out, err := root.OpenFile(storageName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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
