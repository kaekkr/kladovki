package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kaekkr/kladovki/internal/cadastral"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

// The resident's ЖК is found from the premises cadastral number
// (21:319:021:848:8:ВП-2 → ЖК 21:319:021:848:8, unit ВП-2). Ownership itself
// is still confirmed by the ОСИ until the egov (Smart Bridge) check exists.
var (
	ErrCadastralInvalid   = errors.New("Кадастровый номер помещения указан неверно. Пример: 21:319:021:848:8:154")
	ErrCadastralUnknownJK = errors.New("ЖК с таким кадастровым номером ещё не подключён к Kladovki. Проверьте номер или обратитесь в ОСИ.")
	ErrCadastralOtherJK   = errors.New("Помещение с этим кадастровым номером находится в другом ЖК.")
)

// resolvePremises normalizes v.PremisesCadastral and, when it is set, fills the
// premises number and kind from its last facet. It returns the updated
// verification and the ID of the ЖК the premises belongs to ("" when no
// cadastral number was given — older clients send only the premises number).
func (s *Service) resolvePremises(ctx context.Context, v models.Verification) (models.Verification, string, error) {
	number := cadastral.Normalize(v.PremisesCadastral)
	if number == "" {
		return v, "", nil
	}
	complexNo, unit, ok := cadastral.SplitPremises(number)
	if !ok {
		return v, "", ErrCadastralInvalid
	}
	jk, err := s.repo.GetJKByCadastral(ctx, complexNo)
	if errors.Is(err, repository.ErrNotFound) {
		return v, "", ErrCadastralUnknownJK
	}
	if err != nil {
		return v, "", err
	}

	out := v
	out.PremisesCadastral = number
	out.PremisesNumber = unit
	if strings.TrimSpace(out.PremisesKind) == "" {
		out.PremisesKind = cadastral.UnitKind(unit)
	}
	return out, jk.ID, nil
}
