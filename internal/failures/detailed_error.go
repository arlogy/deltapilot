package failures

// DetailedError is designed for cases where additional diagnostic information is needed alongside a regular
// error.
//   - It implements the error interface and can therefore be used wherever a regular error is expected.
//   - It can be recovered from a regular error using errors.AsType() or errors.As() when the additional
//     diagnostic information is needed.
type DetailedError struct {
	PublicCause   error // safe to expose to callers
	InternalCause error // for internal diagnostics; may or may not be sensitive
}

// Error is the only function needed to implement the error interface.
func (e *DetailedError) Error() string {
	return e.PublicCause.Error()
}

// Unwrap exposes the underlying error to errors.Unwrap(), and indirectly to error discovery functions such as
// errors.Is() or errors.As().
func (e *DetailedError) Unwrap() error {
	return e.PublicCause
}
