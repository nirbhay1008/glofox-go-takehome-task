package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}

func WriteError(w http.ResponseWriter, err error) {
	var apiErr *APIError

	if errors.As(err, &apiErr) {
		WriteJSON(w, apiErr.StatusCode, apiErr)
		return
	}

	WriteJSON(w, http.StatusInternalServerError, APIError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "internal server error",
	})
}

func WriteJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(body)
}
