package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glofoxinc/glofox-go-takehome-task/config"
	"github.com/glofoxinc/glofox-go-takehome-task/payment"
	"github.com/glofoxinc/glofox-go-takehome-task/service"
	"github.com/glofoxinc/glofox-go-takehome-task/storage/memory"
)

func TestGetPurchaseInvalidUUID(t *testing.T) {
	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	paymentGateway := payment.AlwaysSucceedingPaymentGateway{}

	purchaseService := service.NewPurchaseService(
		store,
		paymentGateway,
	)

	purchaseHandler := NewPurchaseHandler(purchaseService)

	mux := http.NewServeMux()
	mux.HandleFunc(
		"GET /v1/memberships/purchases/{id}",
		purchaseHandler.GetPurchase,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/memberships/purchases/not-a-uuid",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetPurchaseNotFound(t *testing.T) {
	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	paymentGateway := payment.AlwaysSucceedingPaymentGateway{}

	purchaseService := service.NewPurchaseService(
		store,
		paymentGateway,
	)

	purchaseHandler := NewPurchaseHandler(purchaseService)

	mux := http.NewServeMux()
	mux.HandleFunc(
		"GET /v1/memberships/purchases/{id}",
		purchaseHandler.GetPurchase,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/memberships/purchases/550e8400-e29b-41d4-a716-446655440000",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
