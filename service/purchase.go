package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/glofoxinc/glofox-go-takehome-task/payment"
	"github.com/glofoxinc/glofox-go-takehome-task/storage/memory"
	"github.com/google/uuid"
)

var (
	ErrBlankMemberName          = errors.New("member name is required")
	ErrInvalidStartDate         = errors.New("start date must be in YYYY-MM-DD format")
	ErrPlanNotFound             = errors.New("plan not found")
	ErrInactivePlan             = errors.New("plan is inactive")
	ErrActiveUnlimitedExists    = errors.New("member already has an active unlimited membership")
	ErrPlanSoldOut              = errors.New("plan is sold out")
	ErrPurchaseNotFound         = errors.New("purchase not found")
	ErrPurchaseAlreadyCancelled = errors.New("purchase is already cancelled")
)

type PurchaseService struct {
	store   *memory.Store
	payment payment.PaymentGateway

	// Protects the complete purchase business operation.
	// This ensures two purchases cannot simultaneously consume
	// the same finite inventory.
	mu sync.Mutex
}

func NewPurchaseService(
	store *memory.Store,
	paymentGateway payment.PaymentGateway,
) *PurchaseService {
	return &PurchaseService{
		store:   store,
		payment: paymentGateway,
	}
}

type CreatePurchaseRequest struct {
	MemberName     string
	PlanID         uuid.UUID
	StartDate      time.Time
	IdempotencyKey string
}

func (s *PurchaseService) CreatePurchase(
	req CreatePurchaseRequest,
) (model.Purchase, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if req.IdempotencyKey != "" {
		purchase, exists := s.store.GetPurchaseByIdempotencyKey(req.IdempotencyKey)
		if exists {
			return purchase, nil
		}
	}

	// 1. Normalize and validate member name
	req.MemberName = strings.ToLower(strings.TrimSpace(req.MemberName))

	if req.MemberName == "" {
		return model.Purchase{}, ErrBlankMemberName
	}

	// 2. Find plan
	plan, exists := s.store.GetPlan(req.PlanID)
	if !exists {
		return model.Purchase{}, ErrPlanNotFound
	}

	// 3. Check plan is active
	if !plan.Active {
		return model.Purchase{}, ErrInactivePlan
	}

	// 4. Validate finite inventory
	if plan.RemainingSlots != nil && *plan.RemainingSlots <= 0 {
		return model.Purchase{}, ErrPlanSoldOut
	}

	// 5. Check active unlimited membership rule
	if plan.Type == "unlimited" &&
		s.store.HasActiveUnlimitedMembership(req.MemberName) {
		return model.Purchase{}, ErrActiveUnlimitedExists
	}

	// 6. Charge payment BEFORE changing our state.
	//
	// If payment fails, we simply return the error.
	// No purchase has been created and inventory has not changed.
	paymentResult, err := s.payment.Charge(
		plan.PriceCents,
		req.MemberName,
	)
	if err != nil {
		return model.Purchase{}, err
	}

	// 7. Create purchase and consume inventory atomically
	purchase := model.Purchase{
		ID:            uuid.New(),
		MemberName:    req.MemberName,
		PlanID:        plan.ID,
		PlanName:      plan.Name,
		PlanType:      plan.Type,
		AmountCents:   plan.PriceCents,
		Status:        model.PurchaseStatusActive,
		StartDate:     req.StartDate,
		TransactionID: paymentResult.TransactionID,
	}

	if err := s.store.CompletePurchase(purchase); err != nil {
		if errors.Is(err, memory.ErrPlanSoldOut) {
			return model.Purchase{}, ErrPlanSoldOut
		}

		if errors.Is(err, memory.ErrPlanNotFound) {
			return model.Purchase{}, ErrPlanNotFound
		}

		return model.Purchase{}, err
	}

	if req.IdempotencyKey != "" {
		s.store.SaveIdempotencyKey(req.IdempotencyKey, purchase.ID)
	}

	return purchase, nil
}

func (s *PurchaseService) GetPurchase(
	id uuid.UUID,
) (model.Purchase, error) {
	purchase, exists := s.store.GetPurchase(id)

	if !exists {
		return model.Purchase{}, ErrPurchaseNotFound
	}

	return purchase, nil
}

func (s *PurchaseService) CancelPurchase(id uuid.UUID) (model.Purchase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	purchase, err := s.store.CancelPurchase(id)
	if err != nil {
		if errors.Is(err, memory.ErrPurchaseNotFound) {
			return model.Purchase{}, ErrPurchaseNotFound
		}

		if errors.Is(err, memory.ErrPlanNotFound) {
			return model.Purchase{}, ErrPlanNotFound
		}

		if errors.Is(err, memory.ErrPurchaseAlreadyCancelled) {
			return model.Purchase{}, ErrPurchaseAlreadyCancelled
		}

		return model.Purchase{}, err
	}

	return purchase, nil
}

func (s *PurchaseService) ListPurchases(page, limit int) ([]model.Purchase, bool) {
	return s.store.ListPurchases(page, limit)
}
