package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription is the core domain entity. Fields map directly to the
// subscriptions table.
type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   MonthYear  `json:"start_date"`
	EndDate     *MonthYear `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListFilter is the set of optional filters for the LIST endpoint.
type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

// TotalFilter narrows the total-cost calculation to a [PeriodFrom, PeriodTo]
// window plus optional user_id / service_name filters. Both period bounds are
// inclusive and refer to whole months.
type TotalFilter struct {
	PeriodFrom  MonthYear
	PeriodTo    MonthYear
	UserID      *uuid.UUID
	ServiceName *string
}
