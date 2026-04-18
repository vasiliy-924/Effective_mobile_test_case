package apperrors

import "errors"

var (
	ErrNotFound        = errors.New("subscription not found")
	ErrInvalidArgument = errors.New("invalid argument")
)
