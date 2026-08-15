package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Println("database ready (PostgreSQL)")
	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR(64) PRIMARY KEY,
    role          VARCHAR(50) NOT NULL,
    full_name     VARCHAR(255) NOT NULL,
    phone         VARCHAR(50) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    iin           VARCHAR(20),
    bin           VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    jk_id         VARCHAR(64),
    created_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS jks (
    id         VARCHAR(64) PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    bin        VARCHAR(50) NOT NULL,
    contact    VARCHAR(255) NOT NULL,
    phone      VARCHAR(50) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    owner_id   VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS storages (
    id         VARCHAR(64) PRIMARY KEY,
    jk_id      VARCHAR(64) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
    number     VARCHAR(50) NOT NULL,
    area       NUMERIC(5,2) NOT NULL,
    floor      INT NOT NULL,
    entrance   INT NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(jk_id, number)
);

CREATE TABLE IF NOT EXISTS tariffs (
    id         VARCHAR(64) PRIMARY KEY,
    jk_id      VARCHAR(64) NOT NULL UNIQUE REFERENCES jks(id) ON DELETE CASCADE,
    amount     BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tariff_history (
    id         VARCHAR(64) PRIMARY KEY,
    jk_id      VARCHAR(64) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
    amount     BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS rentals (
    id              VARCHAR(64) PRIMARY KEY,
    storage_id      VARCHAR(64) NOT NULL REFERENCES storages(id) ON DELETE RESTRICT,
    user_id         VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    jk_id           VARCHAR(64) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
    months          INT NOT NULL,
    price_per_month BIGINT NOT NULL,
    total_paid      BIGINT NOT NULL DEFAULT 0,
    starts_at       TIMESTAMPTZ,
    ends_at         TIMESTAMPTZ,
    locked_until    TIMESTAMPTZ,
    status          VARCHAR(50) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    id         VARCHAR(64) PRIMARY KEY,
    rental_id  VARCHAR(64) NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    user_id    VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount     BIGINT NOT NULL,
    provider   VARCHAR(50) NOT NULL DEFAULT 'kaspi',
    status     VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_storages_jk ON storages(jk_id);
`
	_, err := db.Exec(schema)
	return err
}
