package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var errJSONTrailing = errors.New("expected one JSON object")

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errJSONTrailing
	}
	return nil
}

func jsonInput(w http.ResponseWriter, r *http.Request, limit int64, target any, message string) bool {
	if err := decodeJSON(w, r, limit, target); err != nil {
		if errors.Is(err, errJSONTrailing) {
			message = "Ожидается один JSON-объект."
		}
		apiError(w, 400, "invalid_input", message)
		return false
	}
	return true
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
