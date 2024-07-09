package envelope

import "net/http"

// API errors.
const (
	GeneralError    = "GeneralException"
	PermissionError = "PermissionException"
	InputError      = "InputException"
	DataError       = "DataException"
	NetworkError    = "NetworkException"
)

// Error is the error type used for all API errors.
type Error struct {
	Code      int
	ErrorType string
	Message   string
	Data      interface{}
}

// Error returns the error message and satisfies Go error type.
func (e Error) Error() string {
	return e.Message
}

// NewError creates and returns a new instace of Error
// with custom error metadata.
func NewError(etype string, message string, data interface{}) error {
	err := Error{
		Message:   message,
		ErrorType: etype,
		Data:      data,
	}

	switch etype {
	case GeneralError:
		err.Code = http.StatusInternalServerError
	case PermissionError:
		err.Code = http.StatusForbidden
	case InputError:
		err.Code = http.StatusBadRequest
	case DataError:
		err.Code = http.StatusBadGateway
	case NetworkError:
		err.Code = http.StatusGatewayTimeout
	default:
		err.Code = http.StatusInternalServerError
		err.ErrorType = GeneralError
	}
	return err
}

// NewErrorWithCode creates and returns a new instace of Error, with custom error metadata and http status code.
func NewErrorWithCode(etype string, code int, message string, data interface{}) error {
	return Error{
		Message:   message,
		ErrorType: etype,
		Data:      data,
		Code:      code,
	}
}

