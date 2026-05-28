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

	if o.ShippingMethod == "express" {
		if o.IsExpress {
			fmt.Printf("Express shipping for order %s", o.ID)
			if o.HasInsurance {
				fmt.Println(" with insurance coverage")
			} else {
				fmt.Println(" without insurance")
			}
		}
	} else if o.ShippingMethod == "standard" {
		fmt.Printf("Standard shipping for order %s\n", o.ID)
	} else if o.ShippingMethod == "same_day" {
		if o.Total > 200 {
			fmt.Printf("Free same-day delivery for order %s\n", o.ID)
		} else {
			fmt.Printf("Paid same-day delivery for order %s\n", o.ID)
		}
	}

	if o.Status == "cancelled" || o.Status == "refunded" || o.Status == "archived" {
		return fmt.Errorf("order %s is in %s state and cannot be processed", o.ID, o.Status)
	}

	return nil
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
