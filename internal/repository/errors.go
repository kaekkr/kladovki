// internal/repository/errors.go
package repository

import "errors"

var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record already exists")
)
