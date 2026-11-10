package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glofoxinc/glofox-go-takehome-task/api"
	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/glofoxinc/glofox-go-takehome-task/payment"
	"github.com/glofoxinc/glofox-go-takehome-task/service"
	"github.com/glofoxinc/glofox-go-takehome-task/validator"
	"github.com/google/uuid"
)

type PurchaseHandler struct {
	service *service.PurchaseService
}

func NewPurchaseHandler(
	purchaseService *service.PurchaseService,
) *PurchaseHandler {
	return &PurchaseHandler{
		service: purchaseService,
	}
}

func (h *PurchaseHandler) CreatePurchase(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req model.CreatePurchaseRequest

	// 1. Decode HTTP request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_REQUEST",
			Message:    "invalid JSON request",
		})
		return
	}

	// 2. Validate request
	apiErr, err := validator.ValidatePurchaseRequest(req)
	if err != nil {
		api.WriteError(w, &apiErr)
		return
	}

	// Normalize String in Controller
	req.MemberName = strings.ToLower(strings.TrimSpace(req.MemberName))

	// 3. Convert validated strings into domain types
	planID, _ := uuid.Parse(req.PlanID)
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	idempotencyKey := r.Header.Get("Idempotency-Key")

	// 4. Call service
	purchase, err := h.service.CreatePurchase(
		service.CreatePurchaseRequest{
			MemberName:     req.MemberName,
			PlanID:         planID,
			StartDate:      startDate,
			IdempotencyKey: idempotencyKey,
		},
	)

	if err != nil {
		h.handlePurchaseError(w, err)
		return
	}

	// 5. Return HTTP response
	purchaseResponse := toPurchaseResponse(purchase)

	api.WriteJSON(w, http.StatusCreated, purchaseResponse)
}

func (h *PurchaseHandler) GetPurchase(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_PURCHASE_ID",
			Message:    "purchase id must be a valid UUID",
		})
		return
	}

	purchase, err := h.service.GetPurchase(id)
	if err != nil {
		h.handlePurchaseError(w, err)
		return
	}

	purchaseResponse := toPurchaseResponse(purchase)

	api.WriteJSON(w, http.StatusOK, purchaseResponse)
}

func (h *PurchaseHandler) CancelPurchase(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_PURCHASE_ID",
			Message:    "purchase id must be a valid UUID",
		})
		return
	}

	purchase, err := h.service.CancelPurchase(id)
	if err != nil {
		h.handlePurchaseError(w, err)
		return
	}

	purchaseResponse := toPurchaseResponse(purchase)

	api.WriteJSON(w, http.StatusOK, purchaseResponse)
}

func (h *PurchaseHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10

	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			api.WriteError(w, &api.APIError{
				StatusCode: http.StatusBadRequest,
				Code:       "INVALID_PAGE",
				Message:    "page must be a positive integer",
			})
			return
		}

		page = parsed
	}

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			api.WriteError(w, &api.APIError{
				StatusCode: http.StatusBadRequest,
				Code:       "INVALID_LIMIT",
				Message:    "limit must be a positive integer",
			})
			return
		}

		limit = parsed
	}

	purchases, hasMore := h.service.ListPurchases(page, limit)

	response := model.PurchaseListResponse{
		Purchases: make([]model.PurchaseResponse, 0, len(purchases)),
		Page:      page,
		Limit:     limit,
		HasMore:   hasMore,
	}

	for _, purchase := range purchases {
		response.Purchases = append(
			response.Purchases,
			toPurchaseResponse(purchase),
		)
	}

	api.WriteJSON(w, http.StatusOK, response)
}

func (h *PurchaseHandler) handlePurchaseError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, service.ErrBlankMemberName):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_MEMBER_NAME",
			Message:    "member_name is required",
		})

	case errors.Is(err, service.ErrPlanNotFound):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusNotFound,
			Code:       "PLAN_NOT_FOUND",
			Message:    "plan not found",
		})

	case errors.Is(err, service.ErrInactivePlan):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "PLAN_INACTIVE",
			Message:    "plan is inactive",
		})

	case errors.Is(err, service.ErrPlanSoldOut):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusConflict,
			Code:       "PLAN_SOLD_OUT",
			Message:    "plan is sold out",
		})

	case errors.Is(err, service.ErrActiveUnlimitedExists):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusConflict,
			Code:       "ACTIVE_UNLIMITED_EXISTS",
			Message:    "member already has an active unlimited membership",
		})

	case errors.Is(err, service.ErrPurchaseNotFound):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusNotFound,
			Code:       "PURCHASE_NOT_FOUND",
			Message:    "purchase not found",
		})

	case errors.Is(err, service.ErrPurchaseAlreadyCancelled):
		api.WriteError(w, &api.APIError{
			StatusCode: http.StatusConflict,
			Code:       "PURCHASE_ALREADY_CANCELLED",
			Message:    "purchase is already cancelled",
		})

	default:
		var paymentErr *payment.PaymentFailedError

		if errors.As(err, &paymentErr) {
			api.WriteError(w, &api.APIError{
				StatusCode: http.StatusPaymentRequired,
				Code:       "PAYMENT_FAILED",
				Message:    paymentErr.Error(),
			})
			return
		}

		api.WriteError(w, err)
	}
}

func toPurchaseResponse(purchase model.Purchase) model.PurchaseResponse {
	return model.PurchaseResponse{
		ID:            purchase.ID,
		MemberName:    purchase.MemberName,
		PlanID:        purchase.PlanID,
		PlanName:      purchase.PlanName,
		PlanType:      purchase.PlanType,
		AmountCents:   purchase.AmountCents,
		Status:        purchase.Status,
		StartDate:     purchase.StartDate.Format("2006-01-02"),
		TransactionID: purchase.TransactionID,
	}
}
