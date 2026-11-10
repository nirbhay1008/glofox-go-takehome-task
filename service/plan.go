package service

import (
	"errors"

	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/glofoxinc/glofox-go-takehome-task/storage/memory"
	"github.com/google/uuid"
)

var ErrNoPlansFound = errors.New("no plans found")

type PlanService struct {
	store *memory.Store
}

func NewPlanService(store *memory.Store) *PlanService {
	return &PlanService{
		store: store,
	}
}

func (s *PlanService) ListPlans() ([]model.Plan, error) {
	plans := s.store.ListPlans()

	if len(plans) == 0 {
		return []model.Plan{}, ErrNoPlansFound
	}

	return plans, nil
}

func (s *PlanService) GetPlan(id uuid.UUID) (model.Plan, bool) {
	return s.store.GetPlan(id)
}
