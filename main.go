package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/glofoxinc/glofox-go-takehome-task/config"
	"github.com/glofoxinc/glofox-go-takehome-task/handler"
	"github.com/glofoxinc/glofox-go-takehome-task/payment"
	"github.com/glofoxinc/glofox-go-takehome-task/service"
	"github.com/glofoxinc/glofox-go-takehome-task/storage/memory"
)

func main() {
	logSeededPlans()

	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	paymentGateway := payment.AlwaysSucceedingPaymentGateway{}

	purchaseService := service.NewPurchaseService(
		store,
		paymentGateway,
	)
	planService := service.NewPlanService(store)

	mux := http.NewServeMux()
	// Candidates wire their HTTP handlers here.

	healthHandler := handler.NewHealthHandler()
	planHandler := handler.NewPlanHandler(planService)
	purchaseHandler := handler.NewPurchaseHandler(purchaseService)

	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("GET /v1/plans", planHandler.GetPlans)
	mux.HandleFunc("POST /v1/memberships/purchases", purchaseHandler.CreatePurchase)
	mux.HandleFunc("GET /v1/memberships/purchases", purchaseHandler.ListPurchases)
	mux.HandleFunc("GET /v1/memberships/purchases/{id}", purchaseHandler.GetPurchase)
	mux.HandleFunc("POST /v1/memberships/purchases/{id}/cancel", purchaseHandler.CancelPurchase)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func logSeededPlans() {
	log.Println("Seeded membership plans (use these ids in your API):")
	for _, plan := range config.Plans {
		log.Printf(
			"  id=%s name=%q type=%s active=%t price_cents=%d credits=%s remaining_slots=%s",
			plan.ID,
			plan.Name,
			plan.Type,
			plan.Active,
			plan.PriceCents,
			formatOptionalInt(plan.Credits),
			formatOptionalInt(plan.RemainingSlots),
		)
	}
}

func formatOptionalInt(v *int) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *v)
}
