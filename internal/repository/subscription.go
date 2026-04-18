package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wassiliy/subscriptions-service/internal/apperrors"
	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/pkg/month"
)

type Subscription struct {
	pool *pgxpool.Pool
}

func NewSubscription(pool *pgxpool.Pool) *Subscription {
	return &Subscription{pool: pool}
}

func (r *Subscription) Create(ctx context.Context, in domain.CreateSubscription) (domain.Subscription, error) {
	start, err := month.Parse(in.StartDate)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("%w: start_date: %v", apperrors.ErrInvalidArgument, err)
	}
	var endPtr *time.Time
	if in.EndDate != nil && *in.EndDate != "" {
		e, err := month.Parse(*in.EndDate)
		if err != nil {
			return domain.Subscription{}, fmt.Errorf("%w: end_date: %v", apperrors.ErrInvalidArgument, err)
		}
		if e.Before(start) {
			return domain.Subscription{}, fmt.Errorf("%w: end_date before start_date", apperrors.ErrInvalidArgument)
		}
		endPtr = &e
	}

	const q = `
INSERT INTO subscriptions (service_name, price, user_id, start_month, end_month)
VALUES ($1, $2, $3, $4::date, $5::date)
RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, in.ServiceName, in.Price, in.UserID, start, endPtr)
	return scanSubscription(row)
}

func (r *Subscription) GetByID(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	const q = `
SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
FROM subscriptions WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	out, err := scanSubscription(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subscription{}, apperrors.ErrNotFound
		}
		return domain.Subscription{}, err
	}
	return out, nil
}

func (r *Subscription) Update(ctx context.Context, id uuid.UUID, in domain.CreateSubscription) (domain.Subscription, error) {
	start, err := month.Parse(in.StartDate)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("%w: start_date: %v", apperrors.ErrInvalidArgument, err)
	}
	var endPtr *time.Time
	if in.EndDate != nil && *in.EndDate != "" {
		e, err := month.Parse(*in.EndDate)
		if err != nil {
			return domain.Subscription{}, fmt.Errorf("%w: end_date: %v", apperrors.ErrInvalidArgument, err)
		}
		if e.Before(start) {
			return domain.Subscription{}, fmt.Errorf("%w: end_date before start_date", apperrors.ErrInvalidArgument)
		}
		endPtr = &e
	}

	const q = `
UPDATE subscriptions SET
  service_name = $2,
  price = $3,
  user_id = $4,
  start_month = $5::date,
  end_month = $6::date
WHERE id = $1
RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, id, in.ServiceName, in.Price, in.UserID, start, endPtr)
	out, err := scanSubscription(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subscription{}, apperrors.ErrNotFound
		}
		return domain.Subscription{}, err
	}
	return out, nil
}

func (r *Subscription) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM subscriptions WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

func (r *Subscription) List(ctx context.Context, f ListFilter) ([]domain.Subscription, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	args := []any{f.Limit, f.Offset}
	where := "WHERE 1=1"
	if f.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, *f.UserID)
	}
	if f.ServiceName != nil {
		where += fmt.Sprintf(" AND service_name = $%d", len(args)+1)
		args = append(args, *f.ServiceName)
	}

	q := fmt.Sprintf(`
SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
FROM subscriptions
%s
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Subscription
	for rows.Next() {
		s, err := scanSubscriptionRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type CostFilter struct {
	FromMonth time.Time
	ToMonth   time.Time
	UserID    *uuid.UUID
	Service   *string
}

func (r *Subscription) TotalCostForPeriod(ctx context.Context, f CostFilter) (int64, error) {
	if f.ToMonth.Before(f.FromMonth) {
		return 0, fmt.Errorf("%w: period to before from", apperrors.ErrInvalidArgument)
	}

	hasUser := f.UserID != nil
	hasService := f.Service != nil && *f.Service != ""

	const q = `
SELECT COALESCE(SUM(s.price), 0)::bigint
FROM generate_series($1::date, $2::date, interval '1 month') AS gs(m)
JOIN subscriptions s
  ON s.start_month <= (gs.m)::date
 AND (s.end_month IS NULL OR s.end_month >= (gs.m)::date)
WHERE ($3::boolean IS FALSE OR s.user_id = $4::uuid)
  AND ($5::boolean IS FALSE OR s.service_name = $6::text)`

	var uid uuid.UUID
	if hasUser {
		uid = *f.UserID
	}
	var svc string
	if hasService {
		svc = *f.Service
	}

	var total int64
	err := r.pool.QueryRow(ctx, q,
		f.FromMonth, f.ToMonth,
		hasUser, uid,
		hasService, svc,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func scanSubscription(row pgx.Row) (domain.Subscription, error) {
	var (
		id          uuid.UUID
		serviceName string
		price       int
		userID      uuid.UUID
		startMonth  time.Time
		endMonth    *time.Time
		createdAt   time.Time
		updatedAt   time.Time
	)
	if err := row.Scan(&id, &serviceName, &price, &userID, &startMonth, &endMonth, &createdAt, &updatedAt); err != nil {
		return domain.Subscription{}, err
	}
	return toDomain(id, serviceName, price, userID, startMonth, endMonth, createdAt, updatedAt), nil
}

func scanSubscriptionRows(rows pgx.Rows) (domain.Subscription, error) {
	var (
		id          uuid.UUID
		serviceName string
		price       int
		userID      uuid.UUID
		startMonth  time.Time
		endMonth    *time.Time
		createdAt   time.Time
		updatedAt   time.Time
	)
	if err := rows.Scan(&id, &serviceName, &price, &userID, &startMonth, &endMonth, &createdAt, &updatedAt); err != nil {
		return domain.Subscription{}, err
	}
	return toDomain(id, serviceName, price, userID, startMonth, endMonth, createdAt, updatedAt), nil
}

func toDomain(
	id uuid.UUID,
	serviceName string,
	price int,
	userID uuid.UUID,
	startMonth time.Time,
	endMonth *time.Time,
	createdAt, updatedAt time.Time,
) domain.Subscription {
	s := domain.Subscription{
		ID:          id,
		ServiceName: serviceName,
		Price:       price,
		UserID:      userID,
		StartDate:   month.Format(firstOfMonth(startMonth)),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if endMonth != nil {
		em := month.Format(firstOfMonth(*endMonth))
		s.EndDate = &em
	}
	return s
}

func firstOfMonth(t time.Time) time.Time {
	y, m, _ := t.UTC().Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}
