# GIRL Context Pack

**Goal:** Refactor order processing: reduce complexity, flatten nesting, fix ignored errors

**Token budget:** 12000
**Token estimate:** 7800

## Files

- `examples/real-refactors/go-before/main.go`

## Summaries

- `examples/real-refactors/go-before/main.go`: Exports processOrder, main (95 lines, 4 diagnostics, 2 components, 0 hooks)

## SelectedSnippets

### File: main.go (lines 21-96)
```go
func processOrder(o Order) error {
	if o.ID == "" {
		return errors.New("order id is required")
	}
	if o.Total <= 0 {
		return errors.New("total must be positive")
	}
	if len(o.Items) == 0 {
		return errors.New("order must have at least one item")
	}

	discount := 0.0
	if o.Total > 1000 {
		if o.CustomerTier == "gold" {
			discount = o.Total * 0.15
		} else if o.CustomerTier == "silver" {
			if o.Total > 5000 {
				discount = o.Total * 0.12
			} else {
				discount = o.Total * 0.08
			}
		} else {
			if o.Total > 10000 {
				discount = o.Total * 0.10
			}
		}
	}

	totalAfterDiscount := o.Total - discount

	if o.PaymentStatus == "paid" {
		fmt.Printf("Order %s is already paid\n", o.ID)
	} else if o.PaymentStatus == "pending" {
		if o.PaymentMethod == "credit_card" {
			if totalAfterDiscount > 10000 {
				fmt.Println("Processing large credit card payment requiring authorization")
			} else {
				fmt.Println("Processing credit card payment")
			}
		} else if o.PaymentMethod == "paypal" {
			fmt.Println("Processing PayPal payment")
		} else if o.PaymentMethod == "bank_transfer" {
			if o.CustomerTier == "gold" {
				fmt.Println("Processing gold-tier bank transfer with fee waiver")
			} else {
				fmt.Println("Processing standard bank transfer")
			}
		} else {
			_ = fmt.Errorf("unsupported payment method: %s", o.PaymentMethod)
		}
	} else if o.PaymentStatus == "failed" {
		_ = errors.New("payment already marked as failed")
	}
	// ... shipping and status check omitted for budget
```

## Diagnostics

- [HIGH] Function processOrder has cyclomatic complexity 19 (limit: 10)
- [MEDIUM] Function processOrder has nesting depth 4 (limit: 3)
- [HIGH] Function processOrder is 76 lines (limit: 50)
- [MEDIUM] Function processOrder ignores 2 error(s) with _

## Steps

- step_001_go.high-complexity_processOrder: Simplify branching logic in processOrder with guard clauses and early returns
- step_002_go.deep-nesting_processOrder: Reduce nesting depth in processOrder by extracting helper functions
- step_003_go.long-function_processOrder: Extract smaller functions from processOrder
- step_004_go.ignored-error_processOrder: Handle ignored errors in processOrder

## Risks

- Function processOrder has cyclomatic complexity 19 (limit: 10)

## Verification

```bash
go build ./...
go vet ./...
go test ./...
```
