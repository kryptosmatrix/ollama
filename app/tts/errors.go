package tts

import "fmt"

// HTTPError is a user-facing failure with a status the UI layer must honour
// rather than collapsing to 500.
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *HTTPError) HTTPStatus() int {
	if e == nil || e.Code == 0 {
		return 500
	}
	return e.Code
}

func httpErrorf(code int, format string, args ...any) error {
	return &HTTPError{Code: code, Message: fmt.Sprintf(format, args...)}
}
