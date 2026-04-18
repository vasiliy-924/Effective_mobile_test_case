package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/wassiliy/subscriptions-service/internal/domain"
)

// SubscriptionRepository defines persistence for subscriptions.
type SubscriptionRepository interface {
	Create(ctx context.Context, in domain.CreateSubscription) (domain.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Subscription, error)
	Update(ctx context.Context, id uuid.UUID, in domain.CreateSubscription) (domain.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f ListFilter) ([]domain.Subscription, error)
	TotalCostForPeriod(ctx context.Context, f CostFilter) (int64, error)
}
