package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaurya2807/order-processing-system/internal/model"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	const query = `
		INSERT INTO orders (customer_id, status, total_amount, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, customer_id, status, total_amount, created_at`

	order := &model.Order{}
	err := r.db.QueryRow(ctx, query, req.CustomerID, model.StatusPending, req.TotalAmount).
		Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	const query = `
		SELECT id, customer_id, status, total_amount, created_at
		FROM orders
		WHERE id = $1`

	order := &model.Order{}
	err := r.db.QueryRow(ctx, query, id).
		Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("select order %d: %w", id, err)
	}

	return order, nil
}
