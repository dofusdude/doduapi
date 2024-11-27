package main

import (
	"encoding/json"
	"net/http"

	"github.com/charmbracelet/log"
)

var (
	ERR_INVALID_FILTER_VALUE         = "INVALID_FILTER_NAME"
	ERR_INVALID_FILTER_VALUE_MESSAGE = "The filter value you provided is not valid. Please check the format and try again."

	ERR_INVALID_QUERY_VALUE   = "INVALID_QUERY_PARAMETER"
	ERR_INVALID_QUERY_MESSAGE = "The query parameter you provided is not valid. Please check the format and try again."

	ERR_INVALID_JSON_BODY    = "INVALID_JSON_BODY"
	ERR_INVALID_JSON_MESSAGE = "The JSON body you provided is not valid. Please check the format and try again."

	ERR_SERVER_ERROR   = "SERVER_ERROR"
	ERR_SERVER_MESSAGE = "A server error occurred. This is not your fault. Please try again later and contact the administrator."

	ERR_NOT_FOUND         = "NOT_FOUND"
	ERR_NOT_FOUND_MESSAGE = "The requested resource was not found."
)

type ApiError struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func writeNotFoundResponse(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusNotFound, ERR_NOT_FOUND, ERR_NOT_FOUND_MESSAGE, details)
}

func writeServerErrorResponse(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusInternalServerError, ERR_SERVER_ERROR, ERR_SERVER_MESSAGE, details)
}

func writeInvalidFilterResponse(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusBadRequest, ERR_INVALID_FILTER_VALUE, ERR_INVALID_FILTER_VALUE_MESSAGE, details)
}

func writeInvalidQueryResponse(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusBadRequest, ERR_INVALID_QUERY_VALUE, ERR_INVALID_QUERY_MESSAGE, details)
}

func writeInvalidJsonResponse(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusBadRequest, ERR_INVALID_JSON_BODY, ERR_INVALID_JSON_MESSAGE, details)
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message, details string) {
	apiErr := ApiError{
		Status:  status,
		Error:   http.StatusText(status),
		Code:    code,
		Message: message,
		Details: details,
	}

	if status == http.StatusInternalServerError {
		log.Error("Internal Server Error", "code", code, "message", message, "details", details)
	}

	if status == http.StatusBadRequest {
		log.Warn("Bad Request", "code", code, "message", message, "details", details)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiErr)
}
