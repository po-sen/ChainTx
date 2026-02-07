package apperrors

type Type string

const (
	TypeValidation Type = "validation"
	TypeNotFound   Type = "not_found"
	TypeInternal   Type = "internal"
)

type AppError struct {
	Type     Type              `json:"type"`
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func NewInternal(code, message string, metadata map[string]string) *AppError {
	return &AppError{
		Type:     TypeInternal,
		Code:     code,
		Message:  message,
		Metadata: metadata,
	}
}

func NewValidation(code, message string, metadata map[string]string) *AppError {
	return &AppError{
		Type:     TypeValidation,
		Code:     code,
		Message:  message,
		Metadata: metadata,
	}
}
