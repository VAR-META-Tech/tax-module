package domain

import (
	"fmt"
	"net/http"
)

// ErrorCode identifies the category of an application error.
type ErrorCode string

const (
	ErrCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrCodeValidation        ErrorCode = "VALIDATION_ERROR"
	ErrCodeConflict          ErrorCode = "CONFLICT"
	ErrCodeInvalidTransition ErrorCode = "INVALID_STATUS_TRANSITION"
	ErrCodeThirdParty        ErrorCode = "THIRD_PARTY_ERROR"
	ErrCodeTimeout           ErrorCode = "TIMEOUT"
	ErrCodeInternal          ErrorCode = "INTERNAL_ERROR"
	ErrCodeQueueFull         ErrorCode = "QUEUE_FULL"
)

// AppError is the standard error type used across all layers.
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	HTTPStatus int       `json:"-"`
	Err        error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// Constructor helpers

func NewNotFoundError(msg string) *AppError {
	return &AppError{Code: ErrCodeNotFound, Message: msg, HTTPStatus: http.StatusNotFound}
}

func NewValidationError(msg string) *AppError {
	return &AppError{Code: ErrCodeValidation, Message: msg, HTTPStatus: http.StatusBadRequest}
}

func NewConflictError(msg string) *AppError {
	return &AppError{Code: ErrCodeConflict, Message: msg, HTTPStatus: http.StatusConflict}
}

func NewInvalidTransitionError(from, to string) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidTransition,
		Message:    fmt.Sprintf("cannot transition from %s to %s", from, to),
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func NewThirdPartyError(msg string, err error) *AppError {
	return &AppError{Code: ErrCodeThirdParty, Message: msg, HTTPStatus: http.StatusBadGateway, Err: err}
}

func NewTimeoutError(msg string) *AppError {
	return &AppError{Code: ErrCodeTimeout, Message: msg, HTTPStatus: http.StatusGatewayTimeout}
}

func NewInternalError(msg string, err error) *AppError {
	return &AppError{Code: ErrCodeInternal, Message: msg, HTTPStatus: http.StatusInternalServerError, Err: err}
}

func NewQueueFullError() *AppError {
	return &AppError{Code: ErrCodeQueueFull, Message: "job queue is full, try again later", HTTPStatus: http.StatusServiceUnavailable}
}
