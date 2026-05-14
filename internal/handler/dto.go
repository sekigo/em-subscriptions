package handler

import (
	"github.com/google/uuid"

	"github.com/sekigo/em-subscriptions/internal/model"
)

// SubscriptionRequest is the shape of POST / PUT bodies.
// swagger:model SubscriptionRequest
type SubscriptionRequest struct {
	// Name of the subscription service.
	ServiceName string `json:"service_name" example:"Yandex Plus"`
	// Monthly price in rubles (whole rubles, no kopecks).
	Price int `json:"price" example:"400"`
	// User UUID. The service does NOT validate user existence.
	UserID uuid.UUID `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	// Subscription start month, format MM-YYYY.
	StartDate model.MonthYear `json:"start_date" swaggertype:"string" example:"07-2025"`
	// Optional subscription end month, format MM-YYYY.
	EndDate *model.MonthYear `json:"end_date,omitempty" swaggertype:"string" example:"12-2025"`
}

func (r SubscriptionRequest) toModel(id uuid.UUID) *model.Subscription {
	return &model.Subscription{
		ID:          id,
		ServiceName: r.ServiceName,
		Price:       r.Price,
		UserID:      r.UserID,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
	}
}

// SubscriptionResponse is what every endpoint returns for a single record.
// swagger:model SubscriptionResponse
type SubscriptionResponse struct {
	ID          uuid.UUID        `json:"id"`
	ServiceName string           `json:"service_name"`
	Price       int              `json:"price"`
	UserID      uuid.UUID        `json:"user_id"`
	StartDate   model.MonthYear  `json:"start_date" swaggertype:"string" example:"07-2025"`
	EndDate     *model.MonthYear `json:"end_date,omitempty" swaggertype:"string" example:"12-2025"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

func fromModel(s *model.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:          s.ID,
		ServiceName: s.ServiceName,
		Price:       s.Price,
		UserID:      s.UserID,
		StartDate:   s.StartDate,
		EndDate:     s.EndDate,
		CreatedAt:   s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// TotalResponse is returned by /subscriptions/total.
// swagger:model TotalResponse
type TotalResponse struct {
	Total       int64           `json:"total" example:"4800"`
	PeriodFrom  model.MonthYear `json:"period_from" swaggertype:"string" example:"01-2025"`
	PeriodTo    model.MonthYear `json:"period_to" swaggertype:"string" example:"12-2025"`
	UserID      *uuid.UUID      `json:"user_id,omitempty"`
	ServiceName *string         `json:"service_name,omitempty"`
}

// ErrorResponse is the unified shape for non-2xx responses.
// swagger:model ErrorResponse
type ErrorResponse struct {
	Error string `json:"error" example:"validation error: end_date must be on or after start_date"`
}
