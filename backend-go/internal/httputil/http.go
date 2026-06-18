package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const defaultJSONLimit = 1 << 20

type detailError struct {
	Detail string `json:"detail"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func WriteDetailError(w http.ResponseWriter, status int, detail string) {
	WriteJSON(w, status, detailError{Detail: detail})
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, defaultJSONLimit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("body must contain one JSON object")
	}
	return nil
}
