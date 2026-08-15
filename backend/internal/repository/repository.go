package repository

import (
	"database/sql"
)

// Repo управляет подключениями и запросами к БД
type Repo struct {
	db *sql.DB
}

// New создает новый экземпляр репозитория
func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// rowScanner позволяет сканировать как *sql.Row, так и *sql.Rows
type rowScanner interface {
	Scan(dest ...any) error
}
