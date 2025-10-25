package models

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Message      string      `json:"message"`
	Data         interface{} `json:"data,omitempty"`
	TotalResults *int        `json:"total_results,omitempty"`
}

// ErrorResponse is the error response structure
type ErrorResponse struct {
	Message   string      `json:"message"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data"`
}

// NewAPIResponse creates a new API response
func NewAPIResponse(message string, data interface{}) *APIResponse {
	return &APIResponse{
		Message: message,
		Data:    data,
	}
}

// NewAPIResponseWithCount creates a new API response with count
func NewAPIResponseWithCount(message string, data interface{}, count int) *APIResponse {
	return &APIResponse{
		Message:      message,
		Data:         data,
		TotalResults: &count,
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, errorCode string) *ErrorResponse {
	return &ErrorResponse{
		Message:   message,
		ErrorCode: errorCode,
		Data:      nil,
	}
}
