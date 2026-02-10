package apperrors

type Type string

const (
	TypeValidation Type = "validation"
	TypeNotFound   Type = "not_found"
	TypeConflict   Type = "conflict"
	TypeInternal   Type = "internal"
)

type AppError struct {
	Type    Type           `json:"type"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func NewInternal(code, message string, details map[string]any) *AppError {
	return &AppError{
		Type:    TypeInternal,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewValidation(code, message string, details map[string]any) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewNotFound(code, message string, details map[string]any) *AppError {
	return &AppError{
		Type:    TypeNotFound,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewConflict(code, message string, details map[string]any) *AppError {
	return &AppError{
		Type:    TypeConflict,
		Code:    code,
		Message: message,
		Details: details,
	}
}
