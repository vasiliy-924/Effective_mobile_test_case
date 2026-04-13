package domain

import (
	"time"

	"github.com/google/uuid"
)

// Subscription is a persisted subscription row (API + storage shape).
type Subscription struct {
	ID          uuid.UUID `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"` // MM-YYYY
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateSubscription is the request body for create (and fields for update).
type CreateSubscription struct {
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
}

// CostReport is the aggregation response.
type CostReport struct {
	TotalRub    int64      `json:"total_rub"`
	PeriodFrom  string     `json:"period_from"`
	PeriodTo    string     `json:"period_to"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	ServiceName *string    `json:"service_name,omitempty"`
}
