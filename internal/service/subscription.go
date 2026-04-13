package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/pkg/month"
	"github.com/wassiliy/subscriptions-service/internal/repository"
)

type Subscription struct {
	repo   *repository.Subscription
	logger *slog.Logger
}

func NewSubscription(repo *repository.Subscription, logger *slog.Logger) *Subscription {
	return &Subscription{repo: repo, logger: logger}
}

func (s *Subscription) Create(ctx context.Context, in domain.CreateSubscription) (domain.Subscription, error) {
	if err := validateCreate(in); err != nil {
		return domain.Subscription{}, err
	}
	out, err := s.repo.Create(ctx, in)
	if err != nil {
		s.logger.ErrorContext(ctx, "create subscription", slog.Any("err", err))
		return domain.Subscription{}, err
	}
	s.logger.InfoContext(ctx, "subscription created", slog.String("id", out.ID.String()))
	return out, nil
}

func (s *Subscription) Get(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	out, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Subscription{}, err
		}
		s.logger.ErrorContext(ctx, "get subscription", slog.String("id", id.String()), slog.Any("err", err))
	}
	return out, err
}

func (s *Subscription) Update(ctx context.Context, id uuid.UUID, in domain.CreateSubscription) (domain.Subscription, error) {
	if err := validateCreate(in); err != nil {
		return domain.Subscription{}, err
	}
	out, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Subscription{}, err
		}
		s.logger.ErrorContext(ctx, "update subscription", slog.String("id", id.String()), slog.Any("err", err))
	}
	if err == nil {
		s.logger.InfoContext(ctx, "subscription updated", slog.String("id", id.String()))
	}
	return out, err
}

func (s *Subscription) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		s.logger.ErrorContext(ctx, "delete subscription", slog.String("id", id.String()), slog.Any("err", err))
	}
	if err == nil {
		s.logger.InfoContext(ctx, "subscription deleted", slog.String("id", id.String()))
	}
	return err
}

type ListInput struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

func (s *Subscription) List(ctx context.Context, in ListInput) ([]domain.Subscription, error) {
	out, err := s.repo.List(ctx, repository.ListFilter{
		UserID:      in.UserID,
		ServiceName: in.ServiceName,
		Limit:       in.Limit,
		Offset:      in.Offset,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "list subscriptions", slog.Any("err", err))
	}
	return out, err
}

type CostInput struct {
	From, To    string
	UserID      *uuid.UUID
	ServiceName *string
}

func (s *Subscription) Cost(ctx context.Context, in CostInput) (domain.CostReport, error) {
	fromT, err := month.Parse(in.From)
	if err != nil {
		return domain.CostReport{}, fmt.Errorf("%w: from: %v", repository.ErrInvalidArgument, err)
	}
	toT, err := month.Parse(in.To)
	if err != nil {
		return domain.CostReport{}, fmt.Errorf("%w: to: %v", repository.ErrInvalidArgument, err)
	}
	total, err := s.repo.TotalCostForPeriod(ctx, repository.CostFilter{
		FromMonth: fromT,
		ToMonth:   toT,
		UserID:    in.UserID,
		Service:   in.ServiceName,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "subscription cost aggregation", slog.Any("err", err))
		return domain.CostReport{}, err
	}
	s.logger.InfoContext(ctx, "subscription cost computed",
		slog.Int64("total_rub", total),
		slog.String("from", in.From),
		slog.String("to", in.To),
	)
	return domain.CostReport{
		TotalRub:    total,
		PeriodFrom:  in.From,
		PeriodTo:    in.To,
		UserID:      in.UserID,
		ServiceName: in.ServiceName,
	}, nil
}

func validateCreate(in domain.CreateSubscription) error {
	if in.ServiceName == "" {
		return fmt.Errorf("%w: service_name required", repository.ErrInvalidArgument)
	}
	if in.Price < 0 {
		return fmt.Errorf("%w: price must be non-negative", repository.ErrInvalidArgument)
	}
	if in.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id required", repository.ErrInvalidArgument)
	}
	if in.StartDate == "" {
		return fmt.Errorf("%w: start_date required", repository.ErrInvalidArgument)
	}
	return nil
}
