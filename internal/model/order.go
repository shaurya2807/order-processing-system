package model

import "time"

type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusProcessing OrderStatus = "processing"
	StatusCompleted  OrderStatus = "completed"
	StatusCancelled  OrderStatus = "cancelled"
)

type Order struct {
	ID          int64       `json:"id"           db:"id"`
	CustomerID  int64       `json:"customer_id"  db:"customer_id"`
	Status      OrderStatus `json:"status"       db:"status"`
	TotalAmount float64     `json:"total_amount" db:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"   db:"created_at"`
}

type CreateOrderRequest struct {
	CustomerID  int64   `json:"customer_id"  binding:"required"`
	TotalAmount float64 `json:"total_amount" binding:"required,gt=0"`
}
