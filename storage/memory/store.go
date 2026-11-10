package memory

import (
	"errors"
	"sort"
	"sync"

	"github.com/glofoxinc/glofox-go-takehome-task/config"
	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/google/uuid"
)

var (
	ErrPlanNotFound             = errors.New("plan not found")
	ErrPlanSoldOut              = errors.New("plan is sold out")
	ErrPurchaseNotFound         = errors.New("purchase not found")
	ErrPurchaseAlreadyCancelled = errors.New("purchase is already cancelled")
)

type Store struct {
	mu              sync.RWMutex
	plans           map[uuid.UUID]model.Plan
	purchases       map[uuid.UUID]model.Purchase
	idempotencyKeys map[string]uuid.UUID
}

func NewStore() *Store {
	return &Store{
		plans:           make(map[uuid.UUID]model.Plan),
		purchases:       make(map[uuid.UUID]model.Purchase),
		idempotencyKeys: make(map[string]uuid.UUID),
	}
}

func (s *Store) SeedPlans(plans []config.PlanSeed) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, plan := range plans {
		var credits *int
		if plan.Credits != nil {
			value := *plan.Credits
			credits = &value
		}

		var remainingSlots *int
		if plan.RemainingSlots != nil {
			value := *plan.RemainingSlots
			remainingSlots = &value
		}

		s.plans[plan.ID] = model.Plan{
			ID:             plan.ID,
			Name:           plan.Name,
			Type:           plan.Type,
			Active:         plan.Active,
			PriceCents:     plan.PriceCents,
			Credits:        credits,
			RemainingSlots: remainingSlots,
		}
	}
}

func (s *Store) ListPlans() []model.Plan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans := make([]model.Plan, 0, len(s.plans))

	for _, plan := range s.plans {
		plans = append(plans, plan)
	}

	return plans
}

func (s *Store) GetPlan(id uuid.UUID) (model.Plan, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, exists := s.plans[id]
	return plan, exists
}

func (s *Store) CreatePurchase(purchase model.Purchase) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.purchases[purchase.ID] = purchase
}

func (s *Store) GetPurchase(id uuid.UUID) (model.Purchase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	purchase, exists := s.purchases[id]
	return purchase, exists
}

func (s *Store) HasActiveUnlimitedMembership(memberName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, purchase := range s.purchases {
		if purchase.MemberName == memberName &&
			purchase.PlanType == "unlimited" &&
			purchase.Status == model.PurchaseStatusActive {
			return true
		}
	}

	return false
}

func (s *Store) CompletePurchase(purchase model.Purchase) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, exists := s.plans[purchase.PlanID]
	if !exists {
		return ErrPlanNotFound
	}

	// Consume finite inventory.
	if plan.RemainingSlots != nil {
		if *plan.RemainingSlots <= 0 {
			return ErrPlanSoldOut
		}

		remaining := *plan.RemainingSlots - 1
		plan.RemainingSlots = &remaining

		s.plans[plan.ID] = plan
	}

	// Store purchase.
	s.purchases[purchase.ID] = purchase

	return nil
}

func (s *Store) ListPurchases(page, limit int) ([]model.Purchase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	purchases := make([]model.Purchase, 0, len(s.purchases))
	for _, purchase := range s.purchases {
		purchases = append(purchases, purchase)
	}

	sort.Slice(purchases, func(i, j int) bool {
		return purchases[i].ID.String() < purchases[j].ID.String()
	})

	if page < 1 || limit < 1 {
		return []model.Purchase{}, false
	}

	// Prevent integer overflow in (page-1)*limit.
	if page-1 > len(purchases)/limit {
		return []model.Purchase{}, false
	}

	start := (page - 1) * limit

	if start >= len(purchases) {
		return []model.Purchase{}, false
	}

	end := start + limit
	if end > len(purchases) {
		end = len(purchases)
	}

	hasMore := end < len(purchases)

	return purchases[start:end], hasMore
}
func (s *Store) CancelPurchase(id uuid.UUID) (model.Purchase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	purchase, exists := s.purchases[id]
	if !exists {
		return model.Purchase{}, ErrPurchaseNotFound
	}

	if purchase.Status == model.PurchaseStatusCancelled {
		return model.Purchase{}, ErrPurchaseAlreadyCancelled
	}

	plan, exists := s.plans[purchase.PlanID]
	if !exists {
		return model.Purchase{}, ErrPlanNotFound
	}

	// Restore inventory for finite plans
	if plan.RemainingSlots != nil {
		(*plan.RemainingSlots)++
		s.plans[purchase.PlanID] = plan
	}

	// Change status
	purchase.Status = model.PurchaseStatusCancelled
	s.purchases[id] = purchase

	return purchase, nil
}

func (s *Store) GetPurchaseByIdempotencyKey(key string) (model.Purchase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	purchaseID, exists := s.idempotencyKeys[key]
	if !exists {
		return model.Purchase{}, false
	}

	purchase, exists := s.purchases[purchaseID]
	if !exists {
		return model.Purchase{}, false
	}

	return purchase, true
}

func (s *Store) SaveIdempotencyKey(key string, purchaseID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.idempotencyKeys[key] = purchaseID
}
