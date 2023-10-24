package domain

// Error is a package specific error type that lets us define immutable errors.
type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	// ErrBadRequest is returned when the request was invalid.
	ErrBadRequest = Error("invalid request data")

	// ErrInternal is returned when the error is unspecified.
	ErrInternal = Error("internal error")
)
