package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Acceptance records that a user agreed to a version of an agreement.
type Acceptance struct {
	UserID, Document, Version string
	Context                   string // registration | payment | renewal | prompt
	RentalID                  string
	IP, UserAgent             string
}

func (r *Repo) RecordAcceptance(ctx context.Context, a Acceptance) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agreement_acceptances (id, user_id, document, version, context, rental_id, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)`,
		uuid.NewString(), a.UserID, a.Document, a.Version, a.Context, a.RentalID, a.IP, a.UserAgent)
	if err != nil {
		return fmt.Errorf("repository.RecordAcceptance: %w", err)
	}
	return nil
}

func (r *Repo) HasAccepted(ctx context.Context, userID, document, version string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM agreement_acceptances WHERE user_id = $1 AND document = $2 AND version = $3)`,
		userID, document, version).Scan(&ok)
	return ok, err
}
