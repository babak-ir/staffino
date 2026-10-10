package identity

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict = errors.New("conflict")
	ErrValidation = errors.New("validation")
)