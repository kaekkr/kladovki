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

// migrate brings any database to the current schema. Every step is idempotent:
// on a fresh database it recreates the production schema exactly, on the
// existing production database it only applies what is missing.
func migrate(db *sql.DB) error {
	for i, step := range migrations {
		if _, err := db.Exec(step); err != nil {
			return fmt.Errorf("migration step %d: %w", i, err)
		}
	}
	return nil
}

var migrations = []string{
	// --- Base schema (mirrors production as of 2026-11) ---
	`DO $$ BEGIN
		CREATE TYPE user_role AS ENUM ('resident', 'admin');
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN
		CREATE TYPE storage_status AS ENUM ('free', 'occupied', 'locked');
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN
		CREATE TYPE rental_status AS ENUM ('active', 'expired', 'cancelled', 'locked');
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN
		CREATE TYPE payment_status AS ENUM ('pending', 'success', 'failed');
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

	`CREATE TABLE IF NOT EXISTS users (
		id            VARCHAR(36) PRIMARY KEY,
		full_name     VARCHAR(255) NOT NULL,
		phone         VARCHAR(50) NOT NULL UNIQUE,
		email         VARCHAR(255),
		password_hash VARCHAR(255) NOT NULL,
		role          user_role NOT NULL DEFAULT 'resident',
		jk_id         VARCHAR(36),
		created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		iin           VARCHAR(12)
	)`,
	`CREATE TABLE IF NOT EXISTS jks (
		id         VARCHAR(36) PRIMARY KEY,
		name       VARCHAR(255) NOT NULL,
		address    TEXT NOT NULL,
		owner_id   VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	)`,
	`DO $$ BEGIN
		ALTER TABLE users ADD CONSTRAINT fk_users_jk
			FOREIGN KEY (jk_id) REFERENCES jks(id) ON DELETE SET NULL;
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`CREATE TABLE IF NOT EXISTS leads (
		id         SERIAL PRIMARY KEY,
		full_name  VARCHAR(255) NOT NULL,
		phone      VARCHAR(50) NOT NULL,
		email      VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS storages (
		id              VARCHAR(36) PRIMARY KEY,
		jk_id           VARCHAR(36) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
		number          VARCHAR(50) NOT NULL,
		area            NUMERIC(8,2) NOT NULL,
		floor           INT NOT NULL,
		entrance        INT NOT NULL,
		status          storage_status NOT NULL DEFAULT 'free',
		locked_until    TIMESTAMPTZ,
		created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		transfer_to_iin TEXT,
		transfer_until  TIMESTAMPTZ
	)`,
	`CREATE TABLE IF NOT EXISTS tariffs (
		id         VARCHAR(36) PRIMARY KEY,
		jk_id      VARCHAR(36) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
		amount     BIGINT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT tariffs_jk_id_unique UNIQUE (jk_id)
	)`,
	`CREATE TABLE IF NOT EXISTS tariff_history (
		id         VARCHAR(64) PRIMARY KEY,
		jk_id      VARCHAR(64) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
		amount     BIGINT NOT NULL,
		changed_at TIMESTAMPTZ NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS rentals (
		id              VARCHAR(36) PRIMARY KEY,
		storage_id      VARCHAR(36) NOT NULL REFERENCES storages(id) ON DELETE RESTRICT,
		user_id         VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		jk_id           VARCHAR(36) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
		months          INT NOT NULL,
		price_per_month BIGINT NOT NULL,
		total_paid      BIGINT NOT NULL,
		starts_at       TIMESTAMPTZ NOT NULL,
		ends_at         TIMESTAMPTZ NOT NULL,
		status          rental_status NOT NULL DEFAULT 'active',
		created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS payments (
		id         VARCHAR(36) PRIMARY KEY,
		rental_id  VARCHAR(36) NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
		user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		amount     BIGINT NOT NULL,
		provider   VARCHAR(50) NOT NULL,
		status     payment_status NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_storages_jk ON storages(jk_id)`,
	`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_iin ON users(iin)
		WHERE iin IS NOT NULL AND iin <> ''`,

	// --- 2026-11: storage number must be unique inside a JK ---
	// Added only when existing data has no duplicates, so startup never fails;
	// the service layer also checks uniqueness before insert/update.
	`DO $$ BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM storages GROUP BY jk_id, lower(trim(number)) HAVING count(*) > 1
		) THEN
			CREATE UNIQUE INDEX IF NOT EXISTS idx_storages_jk_number
				ON storages (jk_id, lower(trim(number)));
		END IF;
	END $$`,
	`CREATE INDEX IF NOT EXISTS idx_rentals_user ON rentals(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_rentals_storage ON rentals(storage_id)`,
	`CREATE INDEX IF NOT EXISTS idx_payments_rental ON payments(rental_id)`,

	// --- 2026-11: requests (заявки) between tenants and the OSI ---
	`CREATE TABLE IF NOT EXISTS requests (
		id          VARCHAR(36) PRIMARY KEY,
		jk_id       VARCHAR(36) NOT NULL REFERENCES jks(id) ON DELETE CASCADE,
		user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		rental_id   VARCHAR(36) REFERENCES rentals(id) ON DELETE SET NULL,
		type        VARCHAR(32) NOT NULL,
		status      VARCHAR(32) NOT NULL,
		amount      BIGINT NOT NULL DEFAULT 0,
		choice      VARCHAR(16) NOT NULL DEFAULT '',
		old_price   BIGINT NOT NULL DEFAULT 0,
		new_price   BIGINT NOT NULL DEFAULT 0,
		comment     TEXT NOT NULL DEFAULT '',
		resolution  TEXT NOT NULL DEFAULT '',
		created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMPTZ,
		resolved_by VARCHAR(36)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_requests_jk_status ON requests(jk_id, status)`,
	`CREATE INDEX IF NOT EXISTS idx_requests_user ON requests(user_id)`,
}
