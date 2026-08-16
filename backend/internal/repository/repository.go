package repository

import (
	"database/sql"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}
