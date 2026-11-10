package payment

// AlwaysFailingPaymentGateway always returns a payment failure.
// Useful for testing payment failure scenarios.
type AlwaysFailingPaymentGateway struct {
	Message string
}

func (g AlwaysFailingPaymentGateway) Charge(
	amountCents int,
	memberName string,
) (PaymentResult, error) {
	return PaymentResult{}, &PaymentFailedError{
		Message: g.Message,
	}
}
