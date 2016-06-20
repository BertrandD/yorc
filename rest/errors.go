package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Errors

type Errors struct {
	Errors []*Error `json:"errors"`
}

type Error struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func WriteError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(Errors{[]*Error{err}})
}

var (
	ErrNotFound       = &Error{"not_found", 404, "Not Found", "Requested content not found."}
	ErrInternalServer = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)

func NewNotAcceptableError(accept string) *Error {
	return &Error{"not_acceptable", 406, "Not Acceptable", fmt.Sprintf("Accept header must be set to '%s'.", accept)}
}

func NewUnsupportedMediaTypeError(contentType string) *Error {
	return &Error{"unsupported_media_type", 415, "Unsupported Media Type", fmt.Sprintf("Content-Type header must be set to: '%s'.", contentType)}
}
