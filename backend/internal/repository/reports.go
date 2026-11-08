package repository

import (
	"context"
	"fmt"
	"time"
)

// IncomeRow is one money movement: a payment (+) or a refund the ОСИ paid out (−).
type IncomeRow struct {
	At            time.Time
	Kind          string // booking | renewal | refund
	StorageNumber string
	UserName      string
	CompanyName   string
	CompanyBIN    string
	Phone         string
	Amount        int64
}

// IncomeRows lists payments and paid-out refunds of a ЖК in [from, to).
func (r *Repo) IncomeRows(ctx context.Context, jkID string, from, to time.Time) ([]IncomeRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.created_at, p.kind, s.number, u.full_name, u.company_name, u.company_bin, u.phone, p.amount
		FROM payments p
		JOIN rentals rt ON rt.id = p.rental_id
		JOIN storages s ON s.id = rt.storage_id
		JOIN users u ON u.id = p.user_id
		WHERE rt.jk_id = $1 AND p.status = 'success' AND p.created_at >= $2 AND p.created_at < $3

		UNION ALL

		SELECT rq.resolved_at, 'refund', COALESCE(s.number, ''), u.full_name, u.company_name, u.company_bin, u.phone, -rq.amount
		FROM requests rq
		JOIN users u ON u.id = rq.user_id
		LEFT JOIN rentals rt ON rt.id = rq.rental_id
		LEFT JOIN storages s ON s.id = rt.storage_id
		WHERE rq.jk_id = $1 AND `+refundsDoneSQL+` AND rq.resolved_at >= $2 AND rq.resolved_at < $3

		ORDER BY 1`, jkID, from, to)
	if err != nil {
		return nil, fmt.Errorf("repository.IncomeRows: %w", err)
	}
	defer rows.Close()
	var out []IncomeRow
	for rows.Next() {
		var x IncomeRow
		if err := rows.Scan(&x.At, &x.Kind, &x.StorageNumber, &x.UserName, &x.CompanyName, &x.CompanyBIN, &x.Phone, &x.Amount); err != nil {
			return nil, fmt.Errorf("repository.IncomeRows scan: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// StorageRow is one storage with its current tenant (if any).
type StorageRow struct {
	Number      string
	Area        float64
	Floor       int
	Entrance    int
	Status      string
	TenantName  string
	CompanyName string
	Phone       string
	PaidUntil   *time.Time
}

func (r *Repo) StorageRows(ctx context.Context, jkID string) ([]StorageRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.number, s.area, s.floor, s.entrance, s.status::text,
		       COALESCE(u.full_name, ''), COALESCE(u.company_name, ''), COALESCE(u.phone, ''), rt.ends_at
		FROM storages s
		LEFT JOIN rentals rt ON rt.storage_id = s.id AND rt.status = 'active'
		LEFT JOIN users u ON u.id = rt.user_id
		WHERE s.jk_id = $1
		ORDER BY lower(s.number)`, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.StorageRows: %w", err)
	}
	defer rows.Close()
	var out []StorageRow
	for rows.Next() {
		var x StorageRow
		if err := rows.Scan(&x.Number, &x.Area, &x.Floor, &x.Entrance, &x.Status,
			&x.TenantName, &x.CompanyName, &x.Phone, &x.PaidUntil); err != nil {
			return nil, fmt.Errorf("repository.StorageRows scan: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
