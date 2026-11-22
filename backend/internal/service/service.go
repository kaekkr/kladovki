package service

import (
	"errors"

	"github.com/kaekkr/kladovki/internal/notify"
	"github.com/kaekkr/kladovki/internal/repository"
)

var (
	// Same value as the repository's, so handlers recognise "not found" from either layer.
	ErrNotFound           = repository.ErrNotFound
	ErrAlreadyExists      = errors.New("already exists")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalid            = errors.New("invalid")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLocked             = errors.New("storage locked or occupied")
	ErrTransferPending    = errors.New("storage transfer pending")
	ErrPaymentPending     = errors.New("payment already pending")
	// ErrSelfTransfer: the successor's ИИН is the current tenant's own.
	ErrSelfTransfer = errors.New("Нельзя передать кладовку текущему арендатору — укажите ИИН другого жителя.")
	// ErrPreviousTenantDebt: a handover booking while the current tenant is overdue.
	ErrPreviousTenantDebt = errors.New("Предыдущий арендатор не оплатил задолженность за кладовку. Оформить аренду можно после того, как он её погасит.")
	// ErrRefundNotRejectable: the ОСИ may only confirm a tariff refund, not reject it.
	ErrRefundNotRejectable = errors.New("Возврат разницы по тарифу отклонить нельзя: арендатор уже отказался от продления срока. Верните деньги и отметьте «Деньги возвращены».")
	// ErrTerminationPending: renewal refused while an early termination request is open.
	ErrTerminationPending = errors.New("Подана заявка на досрочное расторжение — продление недоступно, пока ОСИ её не рассмотрит.")
	// ErrRenewalPending: early termination refused while a renewal invoice may still be paid.
	ErrRenewalPending = errors.New("Счёт на продление ещё ожидает оплаты в Kaspi. Дождитесь результата оплаты и подайте заявку снова.")
	// ErrTariffChoiceTerminating: a tariff choice refused while an early termination is open.
	ErrTariffChoiceTerminating = errors.New("Подана заявка на досрочное расторжение — перерасчёт по тарифу недоступен, пока ОСИ её не рассмотрит.")
	// ErrTariffChoicePending: early termination refused until the tenant answers the tariff decrease.
	ErrTariffChoicePending = errors.New("Сначала выберите вариант перерасчёта по новому тарифу (продлить срок или вернуть разницу), затем подайте заявку на досрочное расторжение.")
	// ErrRenewalInvoicePending: a renewal invoice was just sent and may still be paid.
	ErrRenewalInvoicePending = errors.New("Счёт на продление уже отправлен в Kaspi. Оплатите его или отправьте новый через 30 минут.")
	ErrStorageInUse          = errors.New("Кладовку с историей аренд удалить нельзя — она нужна для отчётов и споров.")
)

type Service struct {
	repo      *repository.Repo
	notifyCfg notify.Config
	channels  map[string]notify.Channel
}

func New(repo *repository.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Repo() *repository.Repo { return s.repo }
