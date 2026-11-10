package service_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/glofoxinc/glofox-go-takehome-task/config"
	"github.com/glofoxinc/glofox-go-takehome-task/model"
	"github.com/glofoxinc/glofox-go-takehome-task/payment"
	"github.com/glofoxinc/glofox-go-takehome-task/service"
	"github.com/glofoxinc/glofox-go-takehome-task/storage/memory"
	"github.com/google/uuid"
)

func setupService(
	t *testing.T,
	paymentGateway payment.PaymentGateway,
) (*service.PurchaseService, *memory.Store) {
	t.Helper()

	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	purchaseService := service.NewPurchaseService(
		store,
		paymentGateway,
	)

	return purchaseService, store
}

func testDate() time.Time {
	return time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
}

type countingPaymentGateway struct {
	callCount int
}

func (g *countingPaymentGateway) Charge(amountCents int, memberName string) (payment.PaymentResult, error) {
	g.callCount++

	return payment.PaymentResult{
		TransactionID: "txn-test",
	}, nil
}

func TestCreatePurchase_Success(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, store := setupService(t, gateway)

	purchase, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.TenClassPackID,
		StartDate:  testDate(),
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if purchase.ID == uuid.Nil {
		t.Fatal("expected purchase ID to be generated")
	}

	if purchase.MemberName != "John Doe" {
		t.Fatalf(
			"expected member name %q, got %q",
			"John Doe",
			purchase.MemberName,
		)
	}

	if purchase.PlanID != config.TenClassPackID {
		t.Fatalf("unexpected plan ID")
	}

	if purchase.AmountCents != 5000 {
		t.Fatalf(
			"expected amount 5000, got %d",
			purchase.AmountCents,
		)
	}

	if purchase.Status != model.PurchaseStatusActive {
		t.Fatalf(
			"expected status %q, got %q",
			model.PurchaseStatusActive,
			purchase.Status,
		)
	}

	if purchase.TransactionID == "" {
		t.Fatal("expected transaction ID")
	}

	// Verify purchase was actually stored.
	storedPurchase, exists := store.GetPurchase(purchase.ID)

	if !exists {
		t.Fatal("expected purchase to be stored")
	}

	if storedPurchase.ID != purchase.ID {
		t.Fatal("stored purchase ID does not match")
	}
}

func TestCreatePurchase_BlankMemberName(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "   ",
		PlanID:     config.TenClassPackID,
		StartDate:  testDate(),
	})

	if !errors.Is(err, service.ErrBlankMemberName) {
		t.Fatalf(
			"expected ErrBlankMemberName, got %v",
			err,
		)
	}
}

func TestCreatePurchase_PlanNotFound(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     uuid.New(),
		StartDate:  testDate(),
	})

	if !errors.Is(err, service.ErrPlanNotFound) {
		t.Fatalf(
			"expected ErrPlanNotFound, got %v",
			err,
		)
	}
}

func TestCreatePurchase_InactivePlan(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.LegacyPlanID,
		StartDate:  testDate(),
	})

	if !errors.Is(err, service.ErrInactivePlan) {
		t.Fatalf(
			"expected ErrInactivePlan, got %v",
			err,
		)
	}
}

func TestCreatePurchase_CannotHaveMultipleUnlimitedMemberships(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	req := service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.MonthlyUnlimitedID,
		StartDate:  testDate(),
	}

	// First purchase should succeed.
	_, err := svc.CreatePurchase(req)

	if err != nil {
		t.Fatalf(
			"expected first purchase to succeed, got %v",
			err,
		)
	}

	// Second unlimited membership should fail.
	_, err = svc.CreatePurchase(req)

	if !errors.Is(err, service.ErrActiveUnlimitedExists) {
		t.Fatalf(
			"expected ErrActiveUnlimitedExists, got %v",
			err,
		)
	}
}

func TestCreatePurchase_MultiplePackPurchasesAllowed(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	req := service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.TenClassPackID,
		StartDate:  testDate(),
	}

	// Multiple pack purchases should be allowed.
	for i := 0; i < 2; i++ {
		_, err := svc.CreatePurchase(req)

		if err != nil {
			t.Fatalf(
				"expected pack purchase %d to succeed, got %v",
				i+1,
				err,
			)
		}
	}
}

func TestCreatePurchase_LaunchPromoDecreasesInventory(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, store := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.LaunchPromoPackID,
		StartDate:  testDate(),
	})

	if err != nil {
		t.Fatalf(
			"expected purchase to succeed, got %v",
			err,
		)
	}

	plan, exists := store.GetPlan(config.LaunchPromoPackID)

	if !exists {
		t.Fatal("expected launch promo plan to exist")
	}

	if plan.RemainingSlots == nil {
		t.Fatal("expected remaining slots to be present")
	}

	if *plan.RemainingSlots != 4 {
		t.Fatalf(
			"expected 4 remaining slots, got %d",
			*plan.RemainingSlots,
		)
	}
}

func TestCreatePurchase_LaunchPromoCannotOversell(t *testing.T) {
	gateway := payment.AlwaysSucceedingPaymentGateway{}

	svc, _ := setupService(t, gateway)

	// Launch Promo starts with 5 slots.
	for i := 0; i < 5; i++ {
		_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
			MemberName: "Member " + uuid.New().String(),
			PlanID:     config.LaunchPromoPackID,
			StartDate:  testDate(),
		})

		if err != nil {
			t.Fatalf(
				"purchase %d should succeed, got %v",
				i+1,
				err,
			)
		}
	}

	// Sixth purchase must fail.
	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "Member 6",
		PlanID:     config.LaunchPromoPackID,
		StartDate:  testDate(),
	})

	if !errors.Is(err, service.ErrPlanSoldOut) {
		t.Fatalf(
			"expected ErrPlanSoldOut, got %v",
			err,
		)
	}
}

func TestCreatePurchase_PaymentFailure(t *testing.T) {
	gateway := payment.AlwaysFailingPaymentGateway{
		Message: "card declined",
	}

	svc, _ := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.TenClassPackID,
		StartDate:  testDate(),
	})

	var paymentErr *payment.PaymentFailedError

	if !errors.As(err, &paymentErr) {
		t.Fatalf(
			"expected PaymentFailedError, got %v",
			err,
		)
	}
}

func TestCreatePurchase_PaymentFailureDoesNotConsumeInventory(t *testing.T) {
	gateway := payment.AlwaysFailingPaymentGateway{
		Message: "card declined",
	}

	svc, store := setupService(t, gateway)

	_, err := svc.CreatePurchase(service.CreatePurchaseRequest{
		MemberName: "John Doe",
		PlanID:     config.LaunchPromoPackID,
		StartDate:  testDate(),
	})

	if err == nil {
		t.Fatal("expected payment failure")
	}

	plan, exists := store.GetPlan(config.LaunchPromoPackID)

	if !exists {
		t.Fatal("expected launch promo plan to exist")
	}

	if plan.RemainingSlots == nil {
		t.Fatal("expected remaining slots to be present")
	}

	// Payment failed, therefore inventory must remain untouched.
	if *plan.RemainingSlots != 5 {
		t.Fatalf(
			"expected 5 remaining slots, got %d",
			*plan.RemainingSlots,
		)
	}
}

func TestLaunchPromoPackConcurrentPurchases(t *testing.T) {
	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	paymentGateway := payment.AlwaysSucceedingPaymentGateway{}

	purchaseService := service.NewPurchaseService(
		store,
		paymentGateway,
	)

	planID := config.LaunchPromoPackID

	const numberOfRequests = 20

	var wg sync.WaitGroup
	wg.Add(numberOfRequests)

	results := make(chan error, numberOfRequests)

	for i := 0; i < numberOfRequests; i++ {
		go func() {
			defer wg.Done()

			_, err := purchaseService.CreatePurchase(service.CreatePurchaseRequest{
				MemberName: "Member",
				PlanID:     planID,
				StartDate:  time.Now(),
			})

			results <- err
		}()
	}

	wg.Wait()
	close(results)

	successCount := 0
	soldOutCount := 0

	for err := range results {
		if err == nil {
			successCount++
			continue
		}

		if errors.Is(err, service.ErrPlanSoldOut) {
			soldOutCount++
			continue
		}

		t.Fatalf("unexpected error: %v", err)
	}

	if successCount != 5 {
		t.Fatalf("expected 5 successful purchases, got %d", successCount)
	}

	if soldOutCount != 15 {
		t.Fatalf("expected 15 sold-out responses, got %d", soldOutCount)
	}
}

func TestCreatePurchase_IdempotencyKeyReturnsSamePurchase(t *testing.T) {
	store := memory.NewStore()
	store.SeedPlans(config.Plans)

	gateway := &countingPaymentGateway{}

	purchaseService := service.NewPurchaseService(store, gateway)

	planID := config.LaunchPromoPackID

	req := service.CreatePurchaseRequest{
		MemberName:     "Nirbhay",
		PlanID:         planID,
		StartDate:      time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC),
		IdempotencyKey: "purchase-123",
	}

	firstPurchase, err := purchaseService.CreatePurchase(req)
	if err != nil {
		t.Fatalf("first purchase failed: %v", err)
	}

	secondPurchase, err := purchaseService.CreatePurchase(req)
	if err != nil {
		t.Fatalf("second purchase failed: %v", err)
	}

	if firstPurchase.ID != secondPurchase.ID {
		t.Fatalf(
			"expected same purchase ID, got %s and %s",
			firstPurchase.ID,
			secondPurchase.ID,
		)
	}

	if gateway.callCount != 1 {
		t.Fatalf(
			"expected payment gateway to be called once, got %d calls",
			gateway.callCount,
		)
	}
}
