package validator

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/glofoxinc/glofox-go-takehome-task/api"
	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/google/uuid"
)

func ValidatePurchaseRequest(
	req model.CreatePurchaseRequest,
) (api.APIError, error) {

	if strings.TrimSpace(req.MemberName) == "" {
		return api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_MEMBER_NAME",
			Message:    "member_name is required",
		}, errors.New("member name is required")
	}

	if _, err := uuid.Parse(req.PlanID); err != nil {
		return api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_PLAN_ID",
			Message:    "plan_id must be a valid UUID",
		}, err
	}

	if _, err := time.Parse("2006-01-02", req.StartDate); err != nil {
		return api.APIError{
			StatusCode: http.StatusBadRequest,
			Code:       "INVALID_START_DATE",
			Message:    "start_date must be in YYYY-MM-DD format",
		}, err
	}

	return api.APIError{}, nil
}
