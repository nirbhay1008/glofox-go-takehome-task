package model

import "github.com/google/uuid"

type Plan struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Active         bool      `json:"active"`
	PriceCents     int       `json:"price_cents"`
	Credits        *int      `json:"credits"`
	RemainingSlots *int      `json:"remaining_slots"`
}