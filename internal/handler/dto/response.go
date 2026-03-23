package dto

// APIResponse is the standard wrapper for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// Helpers

func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{Success: true, Data: data}
}

func SuccessListResponse(data interface{}, total int64, limit, offset int) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Meta:    &Meta{Total: total, Limit: limit, Offset: offset},
	}
}

func ErrorResponse(code, message string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	}
}
