package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/sekigo/em-subscriptions/internal/model"
	"github.com/sekigo/em-subscriptions/internal/repository"
)

// ErrValidation is returned when the caller supplied logically invalid input
// (e.g. end_date before start_date). Handlers translate it to HTTP 400.
var ErrValidation = errors.New("validation error")

// Repo is the subset of repository methods the service needs. Defined here as
// an interface so the service is trivially mockable in tests.
type Repo interface {
	Create(ctx context.Context, s *model.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
	Update(ctx context.Context, s *model.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f model.ListFilter) ([]model.Subscription, error)
	Total(ctx context.Context, f model.TotalFilter) (int64, error)
}

type SubscriptionService struct {
	repo Repo
}

func New(repo Repo) *SubscriptionService { return &SubscriptionService{repo: repo} }

// ErrNotFound re-exported so handlers don't have to import the repository pkg.
var ErrNotFound = repository.ErrNotFound

func (s *SubscriptionService) Create(ctx context.Context, sub *model.Subscription) error {
	if err := validate(sub); err != nil {
		return err
	}
	return s.repo.Create(ctx, sub)
}

func (s *SubscriptionService) Get(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SubscriptionService) Update(ctx context.Context, sub *model.Subscription) error {
	if err := validate(sub); err != nil {
		return err
	}
	return s.repo.Update(ctx, sub)
}

func (s *SubscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *SubscriptionService) List(ctx context.Context, f model.ListFilter) ([]model.Subscription, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(ctx, f)
}

func (s *SubscriptionService) Total(ctx context.Context, f model.TotalFilter) (int64, error) {
	if f.PeriodFrom.IsZero() || f.PeriodTo.IsZero() {
		return 0, fmt.Errorf("%w: period_from and period_to are required", ErrValidation)
	}
	if f.PeriodTo.Time().Before(f.PeriodFrom.Time()) {
		return 0, fmt.Errorf("%w: period_to must be on or after period_from", ErrValidation)
	}
	return s.repo.Total(ctx, f)
}

func validate(sub *model.Subscription) error {
	if sub.ServiceName == "" {
		return fmt.Errorf("%w: service_name is required", ErrValidation)
	}
	if sub.Price < 0 {
		return fmt.Errorf("%w: price must be >= 0", ErrValidation)
	}
	if sub.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id is required", ErrValidation)
	}
	if sub.StartDate.IsZero() {
		return fmt.Errorf("%w: start_date is required", ErrValidation)
	}
	if sub.EndDate != nil && !sub.EndDate.IsZero() &&
		sub.EndDate.Time().Before(sub.StartDate.Time()) {
		return fmt.Errorf("%w: end_date must be on or after start_date", ErrValidation)
	}
	return nil
}
