package walletkeys

type ErrorCode string

const (
	CodeInvalidKeyMaterialFormat ErrorCode = "invalid_key_material_format"
	CodeInvalidConfiguration     ErrorCode = "invalid_configuration"
	CodeUnsupportedTarget        ErrorCode = "unsupported_allocator_target"
	CodeDerivationFailed         ErrorCode = "address_derivation_failed"
)

type KeyError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *KeyError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func wrapKeyError(code ErrorCode, message string, cause error) *KeyError {
	return &KeyError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
