package repository

import (
	"context"
	"fmt"

	"github.com/kaekkr/kladovki/internal/models"
)

// Debt is the time a tenant keeps a storage after the paid period ended
// (agreement п. 5): every started day costs the rental's monthly price / 30.
// Paid rentals are never expired automatically, so an overdue rental is an
// active one whose ends_at has passed. All fragments use alias "r".
const (
	overdueDaysSQL = `(CASE WHEN r.status = 'active' AND r.ends_at < now()
		THEN CEIL(EXTRACT(EPOCH FROM (now() - r.ends_at)) / 86400)::int ELSE 0 END)`
	debtSQL = `ROUND(r.price_per_month * ` + overdueDaysSQL + ` / 30.0)::bigint`
	// paidRentalSQL excludes bookings that were never paid (holds) — they are not charges.
	paidRentalSQL = `r.status <> 'locked' AND NOT (r.status IN ('expired', 'cancelled') AND r.total_paid = 0)`
)

// ListCharges is the ЖК ledger: what each paid rental has been charged, paid and owes.
func (r *Repo) ListCharges(ctx context.Context, jkID string) ([]*models.Charge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, s.number, u.full_name, u.phone, u.company_name,
		       r.months, r.price_per_month, r.starts_at, r.ends_at, r.total_paid, r.status::text,
		       `+overdueDaysSQL+`, `+debtSQL+`
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1 AND `+paidRentalSQL+`
		ORDER BY `+debtSQL+` DESC, r.ends_at DESC`, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListCharges: %w", err)
	}
	defer rows.Close()

	list := []*models.Charge{}
	for rows.Next() {
		c := &models.Charge{}
		var rentalStatus string
		if err := rows.Scan(&c.RentalID, &c.StorageNumber, &c.UserName, &c.UserPhone, &c.CompanyName,
			&c.Months, &c.PricePerMonth, &c.StartsAt, &c.EndsAt, &c.Paid, &rentalStatus,
			&c.OverdueDays, &c.Debt); err != nil {
			return nil, fmt.Errorf("repository.ListCharges scan: %w", err)
		}
		c.Accrued = c.Paid + c.Debt
		switch {
		case c.Debt > 0:
			c.Status = models.ChargeOverdue
		case rentalStatus == string(models.RentalStatusActive):
			c.Status = models.ChargePaid
		default:
			c.Status = models.ChargeEnded
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// DebtTotals: total debt in the ЖК and how many rentals owe.
func (r *Repo) DebtTotals(ctx context.Context, jkID string) (int64, int, error) {
	var sum int64
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(`+debtSQL+`), 0)::bigint, COUNT(*) FILTER (WHERE `+debtSQL+` > 0)
		FROM rentals r WHERE r.jk_id = $1 AND r.status = 'active'`, jkID).Scan(&sum, &n)
	return sum, n, err
}
