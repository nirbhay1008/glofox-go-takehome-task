package handler

import (
	"errors"
	"net/http"

	"github.com/glofoxinc/glofox-go-takehome-task/api"
	"github.com/glofoxinc/glofox-go-takehome-task/service"
)

type PlanHandler struct {
	service *service.PlanService
}

func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{
		service: planService,
	}
}

func (h *PlanHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.ListPlans()
	if err != nil {
		h.handlePlanError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, plans)
}

func (h *PlanHandler) handlePlanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNoPlansFound):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusNotFound,
			Code:       "NO_PLANS_FOUND",
			Message:    "no plans found",
		})

	default:
		api.WriteError(w, err)
	}
}
