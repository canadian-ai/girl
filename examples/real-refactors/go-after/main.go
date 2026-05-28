package main

import (
	"errors"
	"fmt"
)

type Item struct {
	Name     string
	Price    float64
	Quantity int
}

type Order struct {
	ID             string
	Total          float64
	Items          []Item
	PaymentMethod  string
	PaymentStatus  string
	ShippingMethod string
	IsExpress      bool
	HasInsurance   bool
	Status         string
	CustomerTier   string
}

func validateOrder(o Order) error {
	if o.ID == "" {
		return errors.New("order id is required")
	}
	if o.Total <= 0 {
		return errors.New("total must be positive")
	}
	if len(o.Items) == 0 {
		return errors.New("order must have at least one item")
	}
	return nil
}

func calculateDiscount(total float64, tier string) float64 {
	if total <= 1000 {
		return 0
	}
	switch tier {
	case "gold":
		return total * 0.15
	case "silver":
		if total > 5000 {
			return total * 0.12
		}
		return total * 0.08
	default:
		if total > 10000 {
			return total * 0.10
		}
		return 0
	}
}

func processPayment(status, method string, total float64, tier string) error {
	switch status {
	case "paid":
		return nil
	case "pending":
		break
	case "failed":
		return errors.New("payment already marked as failed")
	default:
		return fmt.Errorf("unknown payment status: %s", status)
	}

	switch method {
	case "credit_card":
		if total > 10000 {
			fmt.Println("Processing large credit card payment requiring authorization")
		} else {
			fmt.Println("Processing credit card payment")
		}
	case "paypal":
		fmt.Println("Processing PayPal payment")
	case "bank_transfer":
		if tier == "gold" {
			fmt.Println("Processing gold-tier bank transfer with fee waiver")
		} else {
			fmt.Println("Processing standard bank transfer")
		}
	default:
		return fmt.Errorf("unsupported payment method: %s", method)
	}
	return nil
}

func processShipping(method string, isExpress, hasInsurance bool, total float64, id string) {
	switch method {
	case "express":
		if !isExpress {
			return
		}
		fmt.Printf("Express shipping for order %s", id)
		if hasInsurance {
			fmt.Println(" with insurance coverage")
		} else {
			fmt.Println(" without insurance")
		}
	case "standard":
		fmt.Printf("Standard shipping for order %s\n", id)
	case "same_day":
		if total > 200 {
			fmt.Printf("Free same-day delivery for order %s\n", id)
		} else {
			fmt.Printf("Paid same-day delivery for order %s\n", id)
		}
	}
}

func checkOrderStatus(status, id string) error {
	if status == "cancelled" || status == "refunded" || status == "archived" {
		return fmt.Errorf("order %s is in %s state and cannot be processed", id, status)
	}
	return nil
}

func processOrder(o Order) error {
	if err := validateOrder(o); err != nil {
		return err
	}

	discount := calculateDiscount(o.Total, o.CustomerTier)
	totalAfterDiscount := o.Total - discount

	if err := processPayment(o.PaymentStatus, o.PaymentMethod, totalAfterDiscount, o.CustomerTier); err != nil {
		return err
	}

	processShipping(o.ShippingMethod, o.IsExpress, o.HasInsurance, o.Total, o.ID)

	return checkOrderStatus(o.Status, o.ID)
}

func main() {
	order := Order{
		ID:             "ORD-001",
		Total:          2500.00,
		Items:          []Item{{Name: "Widget Pro", Price: 2500.00, Quantity: 1}},
		PaymentMethod:  "credit_card",
		PaymentStatus:  "pending",
		ShippingMethod: "express",
		IsExpress:      true,
		HasInsurance:   true,
		Status:         "new",
		CustomerTier:   "silver",
	}
	if err := processOrder(order); err != nil {
		fmt.Printf("Error processing order: %v\n", err)
	}
}
