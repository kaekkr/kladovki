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

	// --- 2026-11: notifications (in-app + delivery queue for external channels) ---
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id VARCHAR(32)`,
	`CREATE TABLE IF NOT EXISTS telegram_link_tokens (
		token      VARCHAR(64) PRIMARY KEY,
		user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS notifications (
		id         VARCHAR(36) PRIMARY KEY,
		user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind       VARCHAR(32) NOT NULL,
		title      TEXT NOT NULL,
		body       TEXT NOT NULL,
		link       TEXT NOT NULL DEFAULT '',
		dedupe_key VARCHAR(200),
		read_at    TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_dedupe ON notifications(user_id, dedupe_key)
		WHERE dedupe_key IS NOT NULL`,
	`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC)`,
	`CREATE TABLE IF NOT EXISTS notification_deliveries (
		id              VARCHAR(36) PRIMARY KEY,
		notification_id VARCHAR(36) NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
		channel         VARCHAR(16) NOT NULL,
		status          VARCHAR(16) NOT NULL DEFAULT 'pending',
		attempts        INT NOT NULL DEFAULT 0,
		last_error      TEXT NOT NULL DEFAULT '',
		next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sent_at         TIMESTAMPTZ,
		created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_deliveries_pending ON notification_deliveries(next_attempt_at)
		WHERE status = 'pending'`,

	// Renewal payments extend an active rental by `months`; booking payments activate a hold.
	`ALTER TABLE payments ADD COLUMN IF NOT EXISTS kind VARCHAR(16) NOT NULL DEFAULT 'booking'`,
	`ALTER TABLE payments ADD COLUMN IF NOT EXISTS months INT NOT NULL DEFAULT 0`,
	// How long a booking is held before payment (ОСИ setting).
	`ALTER TABLE jks ADD COLUMN IF NOT EXISTS lock_minutes INT NOT NULL DEFAULT 2`,

	// ОСИ approval of new users (only owners may rent). Existing users stay active.
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active'`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_type VARCHAR(16) NOT NULL DEFAULT 'person'`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS premises_kind VARCHAR(16) NOT NULL DEFAULT 'flat'`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS premises_number VARCHAR(50) NOT NULL DEFAULT ''`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS company_bin VARCHAR(12) NOT NULL DEFAULT ''`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS company_name VARCHAR(255) NOT NULL DEFAULT ''`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS review_comment TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(36)`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ`,
	`CREATE INDEX IF NOT EXISTS idx_users_jk_status ON users(jk_id, status)`,

	// Proof of consent to the user agreement: who accepted which version, when, where.
	`CREATE TABLE IF NOT EXISTS agreement_acceptances (
		id          VARCHAR(36) PRIMARY KEY,
		user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		document    VARCHAR(32) NOT NULL,
		version     VARCHAR(32) NOT NULL,
		context     VARCHAR(16) NOT NULL,
		rental_id   VARCHAR(36),
		ip          VARCHAR(64) NOT NULL DEFAULT '',
		user_agent  TEXT NOT NULL DEFAULT '',
		accepted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_acceptances_user ON agreement_acceptances(user_id, document, version)`,

	// Record of a forced termination for debt (agreement п. 5.2, ОСИ regulation п. 4.2–4.4).
	`CREATE TABLE IF NOT EXISTS terminations (
		id         VARCHAR(36) PRIMARY KEY,
		rental_id  VARCHAR(36) NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
		reason     TEXT NOT NULL,
		note       TEXT NOT NULL DEFAULT '',
		police     BOOLEAN NOT NULL DEFAULT false,
		costs      BIGINT NOT NULL DEFAULT 0,
		created_by VARCHAR(36),
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_terminations_rental ON terminations(rental_id)`,

	// The expiry worker scans by (status, ends_at) every 10 s; dashboards filter by ЖК.
	`CREATE INDEX IF NOT EXISTS idx_rentals_status_ends ON rentals(status, ends_at)`,
	`CREATE INDEX IF NOT EXISTS idx_rentals_jk ON rentals(jk_id)`,
}
