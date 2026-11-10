package model

import (
	"github.com/google/uuid"
)

type PurchaseResponse struct {
	ID            uuid.UUID      `json:"id"`
	MemberName    string         `json:"member_name"`
	PlanID        uuid.UUID      `json:"plan_id"`
	PlanName      string         `json:"plan_name"`
	PlanType      string         `json:"plan_type"`
	AmountCents   int            `json:"amount_cents"`
	Status        PurchaseStatus `json:"status"`
	StartDate     string         `json:"start_date"`
	TransactionID string         `json:"transaction_id"`
}

type PurchaseListResponse struct {
	Purchases []PurchaseResponse `json:"purchases"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
	HasMore   bool               `json:"has_more"`
}
