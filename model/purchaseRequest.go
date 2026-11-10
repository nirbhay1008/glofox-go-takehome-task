package model

type CreatePurchaseRequest struct {
	MemberName string `json:"member_name"`
	PlanID     string `json:"plan_id"`
	StartDate  string `json:"start_date"`
}
