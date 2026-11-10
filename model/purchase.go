package model

import (
	"time"

	"github.com/google/uuid"
)

type PurchaseStatus string

const (
	PurchaseStatusActive    PurchaseStatus = "active"
	PurchaseStatusCancelled PurchaseStatus = "cancelled"
)

type Purchase struct {
	ID            uuid.UUID
	MemberName    string
	PlanID        uuid.UUID
	PlanName      string
	PlanType      string
	AmountCents   int
	Status        PurchaseStatus
	StartDate     time.Time
	TransactionID string
}
