package repository

import "errors"

var (
	// ErrNotFound is returned when a requested record does not exist in the database.
	ErrNotFound = errors.New("record not found")

	// ErrConflict is returned when attempting to insert a record that violates a unique constraint.
	ErrConflict = errors.New("record already exists")

	// ErrStorageAlreadyLocked is returned when attempting to lock or rent a storage that is already locked or occupied.
	ErrStorageAlreadyLocked = errors.New("storage is already locked or rented by another user")
)
