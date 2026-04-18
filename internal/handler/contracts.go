package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/service"
)

// SubscriptionService is the application API used by HTTP handlers.
type SubscriptionService interface {
	Create(ctx context.Context, in domain.CreateSubscription) (domain.Subscription, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Subscription, error)
	Update(ctx context.Context, id uuid.UUID, in domain.CreateSubscription) (domain.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, in service.ListInput) ([]domain.Subscription, error)
	Cost(ctx context.Context, in service.CostInput) (domain.CostReport, error)
}
