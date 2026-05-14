package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sekigo/em-subscriptions/internal/model"
)

var ErrNotFound = errors.New("subscription not found")

type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

const columns = `id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

func (r *SubscriptionRepo) Create(ctx context.Context, s *model.Subscription) error {
	const q = `
INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING ` + columns

	row := r.pool.QueryRow(ctx, q,
		s.ServiceName, s.Price, s.UserID, s.StartDate, nullableMonthYear(s.EndDate),
	)
	return scanSubscription(row, s)
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	const q = `SELECT ` + columns + ` FROM subscriptions WHERE id = $1`

	var s model.Subscription
	err := scanSubscription(r.pool.QueryRow(ctx, q, id), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return &s, nil
}

func (r *SubscriptionRepo) Update(ctx context.Context, s *model.Subscription) error {
	const q = `
UPDATE subscriptions
SET service_name = $2,
    price        = $3,
    user_id      = $4,
    start_date   = $5,
    end_date     = $6,
    updated_at   = NOW()
WHERE id = $1
RETURNING ` + columns

	row := r.pool.QueryRow(ctx, q,
		s.ID, s.ServiceName, s.Price, s.UserID, s.StartDate, nullableMonthYear(s.EndDate),
	)
	err := scanSubscription(row, s)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (r *SubscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM subscriptions WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepo) List(ctx context.Context, f model.ListFilter) ([]model.Subscription, error) {
	const q = `
SELECT ` + columns + `
FROM subscriptions
WHERE ($1::uuid IS NULL OR user_id      = $1)
  AND ($2::text IS NULL OR service_name = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, q, f.UserID, f.ServiceName, f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var out []model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := scanSubscription(rows, &s); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Total returns the aggregate cost over [PeriodFrom, PeriodTo].
//
// For each subscription overlapping the window we count whole months between
//   lower = max(start_date, period_from)
//   upper = min(end_date OR period_to, period_to)
// inclusive of both ends, then multiply by the monthly price.
func (r *SubscriptionRepo) Total(ctx context.Context, f model.TotalFilter) (int64, error) {
	const q = `
WITH bounds AS (
    SELECT
        GREATEST(s.start_date, $1::date)                              AS lower_bound,
        LEAST(COALESCE(s.end_date, $2::date), $2::date)               AS upper_bound,
        s.price
    FROM subscriptions s
    WHERE s.start_date <= $2::date
      AND (s.end_date IS NULL OR s.end_date >= $1::date)
      AND ($3::uuid IS NULL OR s.user_id      = $3)
      AND ($4::text IS NULL OR s.service_name = $4)
)
SELECT COALESCE(SUM(
    price * (
        (EXTRACT(YEAR FROM upper_bound)::int  - EXTRACT(YEAR FROM lower_bound)::int)  * 12
      + (EXTRACT(MONTH FROM upper_bound)::int - EXTRACT(MONTH FROM lower_bound)::int)
      + 1
    )
), 0)::bigint
FROM bounds`

	var total int64
	err := r.pool.QueryRow(ctx, q,
		f.PeriodFrom, f.PeriodTo, f.UserID, f.ServiceName,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total subscriptions: %w", err)
	}
	return total, nil
}

// --- helpers ------------------------------------------------------------

// row is the minimal surface we need from pgx.Row and pgx.Rows so we can scan
// both single rows and rows-iterator items with the same helper.
type row interface {
	Scan(dest ...any) error
}

func scanSubscription(r row, s *model.Subscription) error {
	var end *model.MonthYear
	if err := r.Scan(
		&s.ID, &s.ServiceName, &s.Price, &s.UserID,
		&s.StartDate, &end, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return err
	}
	s.EndDate = end
	return nil
}

// nullableMonthYear turns a *MonthYear into a value pgx can bind: either a
// concrete MonthYear (which implements driver.Valuer) or untyped nil.
func nullableMonthYear(m *model.MonthYear) any {
	if m == nil || m.IsZero() {
		return nil
	}
	return *m
}
