package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

type DashboardStatsRow struct {
	Received         int64 // net of refunds paid out in the period
	PreviousReceived int64
	Refunded         int64

	Debt           int64
	DebtApartments int

	OccupiedStorages int
	TotalStorages    int

	ActiveRentals int
	NewRentals    int
}

type RevenuePointRow struct {
	Date   time.Time
	Amount int64
}

func (r *Repo) GetDashboardStats(
	ctx context.Context,
	jkID string,
	from time.Time,
	to time.Time,
	previousFrom time.Time,
	previousTo time.Time,
) (*DashboardStatsRow, error) {
	const query = `
	WITH current_payments AS (
		SELECT COALESCE(SUM(p.amount), 0) AS amount
		FROM payments p
		INNER JOIN rentals r ON r.id = p.rental_id
		WHERE r.jk_id = $1
		  AND p.status = 'success'
		  AND p.created_at >= $2
		  AND p.created_at < $3
	),

	current_refunds AS (
		SELECT COALESCE(SUM(rq.amount), 0) AS amount
		FROM requests rq
		WHERE rq.jk_id = $1 AND ` + refundsDoneSQL + `
		  AND rq.resolved_at >= $2
		  -- current period runs up to now; no upper bound (DB and app clocks differ by ms)
	),

	previous_refunds AS (
		SELECT COALESCE(SUM(rq.amount), 0) AS amount
		FROM requests rq
		WHERE rq.jk_id = $1 AND ` + refundsDoneSQL + `
		  AND rq.resolved_at >= $4
		  AND rq.resolved_at < $5
	),

	previous_payments AS (
		SELECT COALESCE(SUM(p.amount), 0) AS amount
		FROM payments p
		INNER JOIN rentals r ON r.id = p.rental_id
		WHERE r.jk_id = $1
		  AND p.status = 'success'
		  AND p.created_at >= $4
		  AND p.created_at < $5
	),

	storage_stats AS (
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (
				WHERE status = 'occupied'
			) AS occupied
		FROM storages
		WHERE jk_id = $1
	),

	rental_stats AS (
		SELECT
			COUNT(*) FILTER (
				WHERE status = 'active'
			) AS active,

			COUNT(*) FILTER (
				WHERE status = 'active'
				  AND created_at >= $2
				  AND created_at < $3
			) AS new_rentals
		FROM rentals
		WHERE jk_id = $1
	)

	SELECT
		(SELECT amount FROM current_payments) - (SELECT amount FROM current_refunds),
		(SELECT amount FROM previous_payments) - (SELECT amount FROM previous_refunds),
		(SELECT amount FROM current_refunds),

		(SELECT occupied FROM storage_stats),
		(SELECT total FROM storage_stats),

		(SELECT active FROM rental_stats),
		(SELECT new_rentals FROM rental_stats)
`
	var stats DashboardStatsRow

	err := r.db.QueryRowContext(
		ctx,
		query,
		jkID,
		from,
		to,
		previousFrom,
		previousTo,
	).Scan(
		&stats.Received,
		&stats.PreviousReceived,
		&stats.Refunded,

		&stats.OccupiedStorages,
		&stats.TotalStorages,

		&stats.ActiveRentals,
		&stats.NewRentals,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.GetDashboardStats: %w", err)
	}

	return &stats, nil
}

func (r *Repo) GetDashboardRevenue(
	ctx context.Context,
	jkID string,
	from time.Time,
	to time.Time,
	groupBy string,
) ([]RevenuePointRow, error) {
	query := fmt.Sprintf(`
		SELECT
			-- Kazakhstan days (UTC+5 since 2024); explicit offset, independent of tzdata
			DATE_TRUNC('%s', (t.at AT TIME ZONE 'UTC') + interval '5 hours') AS period,
			COALESCE(SUM(t.amount), 0) AS amount
		FROM (
			SELECT p.created_at AS at, p.amount
			FROM payments p
			INNER JOIN rentals r ON r.id = p.rental_id
			WHERE r.jk_id = $1
			  AND p.status = 'success'
			  AND p.created_at >= $2
			  AND p.created_at < $3

			UNION ALL

			-- refunds paid out by the ОСИ reduce income on the day they were made
			SELECT rq.resolved_at, -rq.amount
			FROM requests rq
			WHERE rq.jk_id = $1 AND `+refundsDoneSQL+`
			  AND rq.resolved_at >= $2
			  AND rq.resolved_at < $3 + interval '1 minute'
		) t
		GROUP BY period
		ORDER BY period ASC
	`, groupBy)

	rows, err := r.db.QueryContext(
		ctx,
		query,
		jkID,
		from,
		to,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.GetDashboardRevenue query: %w", err)
	}
	defer rows.Close()

	var result []RevenuePointRow

	for rows.Next() {
		var row RevenuePointRow

		if err := rows.Scan(&row.Date, &row.Amount); err != nil {
			return nil, fmt.Errorf("repository.GetDashboardRevenue scan: %w", err)
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.GetDashboardRevenue rows: %w", err)
	}

	return result, nil
}

func (r *Repo) ListDashboardAttention(
	ctx context.Context,
	jkID string,
) ([]*models.DashboardAttention, error) {
	query := `
		SELECT
			'rental_overdue' AS type,
			r.id AS rental_id,
			s.id AS storage_id,
			s.number AS storage_number,
			u.id AS user_id,
			u.full_name,
			u.phone,
			r.ends_at,
			0 AS days_left,
			` + overdueDaysSQL + ` AS days_overdue,
			` + debtSQL + ` AS debt,
			0 AS sort_group
		FROM rentals r
		INNER JOIN storages s ON s.id = r.storage_id
		INNER JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1
		  AND r.status = 'active'
		  AND r.ends_at <= NOW()

		UNION ALL

		SELECT
			'rental_expiring' AS type,
			r.id AS rental_id,
			s.id AS storage_id,
			s.number AS storage_number,
			u.id AS user_id,
			u.full_name,
			u.phone,
			r.ends_at,
			GREATEST(
				CEIL(
					EXTRACT(EPOCH FROM (r.ends_at - NOW())) / 86400
				),
				0
			)::int AS days_left,
			0 AS days_overdue,
			0::bigint AS debt,
			1 AS sort_group
		FROM rentals r
		INNER JOIN storages s ON s.id = r.storage_id
		INNER JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1
		  AND r.status = 'active'
		  AND r.ends_at > NOW()
		  AND r.ends_at <= NOW() + INTERVAL '7 days'

		UNION ALL

		SELECT
			'storage_locked' AS type,
			r.id AS rental_id,
			s.id AS storage_id,
			s.number AS storage_number,
			u.id AS user_id,
			u.full_name,
			u.phone,
			r.ends_at,
			0 AS days_left,
			0 AS days_overdue,
			0::bigint AS debt,
			2 AS sort_group
		FROM rentals r
		INNER JOIN storages s ON s.id = r.storage_id
		INNER JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1
		  AND r.status = 'locked'
		  AND r.ends_at > NOW()

		ORDER BY sort_group ASC, ends_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.ListDashboardAttention query: %w",
			err,
		)
	}
	defer rows.Close()

	list := make([]*models.DashboardAttention, 0)

	for rows.Next() {
		var item models.DashboardAttention

		if err := rows.Scan(
			&item.Type,
			&item.RentalID,
			&item.StorageID,
			&item.StorageNumber,
			&item.UserID,
			&item.FullName,
			&item.Phone,
			&item.EndsAt,
			&item.DaysLeft,
			&item.DaysOverdue,
			&item.Debt,
			new(int),
		); err != nil {
			return nil, fmt.Errorf(
				"repository.ListDashboardAttention scan: %w",
				err,
			)
		}

		list = append(list, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"repository.ListDashboardAttention rows: %w",
			err,
		)
	}

	return list, nil
}

func (r *Repo) ListDashboardActivity(
	ctx context.Context,
	jkID string,
	limit int,
) ([]*models.DashboardActivity, error) {
	const query = `
		SELECT
			p.id,
			'payment' AS type,
			u.id AS user_id,
			u.full_name,
			s.number AS storage_number,
			p.amount,
			p.rental_id,
			p.created_at
		FROM payments p
		INNER JOIN rentals r ON r.id = p.rental_id
		INNER JOIN users u ON u.id = p.user_id
		INNER JOIN storages s ON s.id = r.storage_id
		WHERE r.jk_id = $1
		  AND p.status = 'success'

		ORDER BY p.created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, jkID, limit)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.ListDashboardActivity query: %w",
			err,
		)
	}
	defer rows.Close()

	list := make([]*models.DashboardActivity, 0)

	for rows.Next() {
		var item models.DashboardActivity

		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.UserID,
			&item.FullName,
			&item.StorageNumber,
			&item.Amount,
			&item.RentalID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"repository.ListDashboardActivity scan: %w",
				err,
			)
		}

		list = append(list, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"repository.ListDashboardActivity rows: %w",
			err,
		)
	}

	return list, nil
}

func (r *Repo) GetDashboardOccupancy(
	ctx context.Context,
	jkID string,
) (*models.DashboardOccupancy, error) {
	const query = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'free') AS free,
			COUNT(*) FILTER (WHERE status = 'occupied') AS occupied,
			COUNT(*) FILTER (WHERE status = 'locked') AS locked
		FROM storages
		WHERE jk_id = $1
	`

	var occupancy models.DashboardOccupancy

	err := r.db.QueryRowContext(
		ctx,
		query,
		jkID,
	).Scan(
		&occupancy.Total,
		&occupancy.Free,
		&occupancy.Occupied,
		&occupancy.Locked,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.GetDashboardOccupancy: %w",
			err,
		)
	}

	return &occupancy, nil
}

// refundsDoneSQL: requests whose money the ОСИ has returned (alias "rq").
const refundsDoneSQL = `rq.status = 'done' AND (rq.type IN ('payment_refund', 'early_termination')
	OR (rq.type = 'tariff_recalc' AND rq.choice = 'refund'))`
