# Submission notes

## Time spent

Approximately: 3–3.5 hours

## Assumptions

- Kept the implementation focused on the requirements and used in-memory storage as specified.
- Used `member_name` to identify members, since a member entity/ID was not part of the provided API.
- An unlimited plan can be purchased again if the member's previous unlimited membership is no longer active (for example, cancelled).
- Kept the API responses simple and focused on the information needed by the client.
- Member names are normalized by trimming leading/trailing whitespace and converting to lowercase. Therefore, names such as Nirbhay, nirbhay, and NIRBHAY are treated as the same member.

## API / design choices

- Kept the purchase API as the main focus, covering validation, inactive plans, duplicate unlimited memberships, finite inventory, payment failures, and concurrent purchases.
- Used a layered structure with handlers, services, and storage to keep responsibilities separated.
- Added idempotency support, purchase cancellation, and pagination as additional improvements.

## Tests

What I covered and why:

- Purchase creation — verifies the main happy path.
- Request validation — verifies invalid/blank inputs are handled correctly.
- Inactive and unknown plans — ensures invalid plans cannot be purchased.
- Unlimited membership rule — ensures a member cannot have multiple active unlimited memberships.
- Promo inventory — verifies inventory is reduced correctly and cannot be oversold.
- Payment failures — verifies that failed payments do not create purchases or consume inventory.
- Concurrent purchases — verifies the finite inventory remains consistent under concurrent requests.
- Idempotency — verifies that retrying with the same idempotency key does not create another purchase or charge the payment gateway again.
- Handler/API tests — verifies important HTTP-level error responses such as invalid and unknown purchase IDs.

## Known gaps / what I would do next

- Authentication/authorization is not implemented for the APIs.
- ⁠The current purchase flow processes payment before persisting the purchase and decrementing inventory. This keeps the implementation simple and ensures failed payments do not consume
inventory. However, if payment succeeds but purchase persistence fails, the customer could be charged without receiving a confirmed purchase.
In a production system, I would introduce an inventory reservation mechanism with an expiry time. The flow would be: validate request → reserve inventory → process payment using an idempotency key → confirm the purchase and finalize inventory on success, or release the reservation on failure. I would also implement payment reconciliation to handle uncertain payment outcomes and persistence failures.
- Purchases are identified using `member_name` rather than a `member_id`. In a production system, I would use a member ID and have a separate member entity.
- The purchase flow currently holds the service lock while waiting for the payment gateway. In a production system, I would consider making the payment operation asynchronous or using a more fine-grained concurrency approach.
- Pagination currently sorts purchases by UUID to make results deterministic; a production implementation would ideally paginate using a meaningful field such as creation time.
- Idempotency keys currently return the original purchase when reused, even if the retry contains different purchase details. A production implementation should store the original request details with the idempotency key and reject reuse with a different payload.

## AI tools used (optional)

- ChatGPT — used for guidance, debugging, and discussing design/testing approaches. The implementation and final decisions were reviewed and adapted as part of the submission.