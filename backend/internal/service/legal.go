package service

import (
	"context"
	"errors"
	"log"

	"github.com/kaekkr/kladovki/internal/legal"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

var ErrTermsNotAccepted = errors.New("Примите Пользовательское соглашение и правила аренды кладовых.")

// TermsAccepted reports whether the client sent the current version of a document.
func TermsAccepted(doc, version string) bool {
	return version != "" && version == legal.Version(doc)
}

// RecordAcceptance stores proof of consent. Failures are logged, not fatal:
// the action the user consented to has already happened.
func (s *Service) RecordAcceptance(ctx context.Context, a repository.Acceptance) {
	if err := s.repo.RecordAcceptance(ctx, a); err != nil {
		log.Printf("record acceptance %s/%s for %s: %v", a.Document, a.Version, a.UserID, err)
	}
}

// AgreementStatus: per document, its current version and whether the user accepted it.
type AgreementStatus struct {
	Version  string `json:"version"`
	Accepted bool   `json:"accepted"`
}

func (s *Service) LegalStatus(ctx context.Context, userID string) (map[string]AgreementStatus, error) {
	out := map[string]AgreementStatus{}
	for _, doc := range []string{legal.ResidentTerms, legal.OSIRules} {
		v := legal.Version(doc)
		ok, err := s.repo.HasAccepted(ctx, userID, doc, v)
		if err != nil {
			return nil, err
		}
		out[doc] = AgreementStatus{Version: v, Accepted: ok}
	}
	return out, nil
}

// DocumentForRole: residents accept the resident agreement, ОСИ admins the ОСИ regulation.
func DocumentForRole(role models.Role) string {
	if role == models.RoleAdmin {
		return legal.OSIRules
	}
	return legal.ResidentTerms
}
