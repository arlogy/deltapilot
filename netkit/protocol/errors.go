package protocol

import (
	"errors"
)

var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrMessageIncomplete = errors.New("message incomplete")
	ErrMessageTooLarge   = errors.New("message too large")
	ErrTimeout           = errors.New("timeout")

	ErrReadFailed  = errors.New("read failed")
	ErrWriteFailed = errors.New("write failed")
)
