package service

import (
	"context"
	"fmt"

	"github.com/shaurya2807/order-processing-system/internal/model"
	"github.com/shaurya2807/order-processing-system/internal/repository"
	"go.uber.org/zap"
)

type OrderService struct {
	repo   *repository.OrderRepository
	logger *zap.Logger
}

func NewOrderService(repo *repository.OrderRepository, logger *zap.Logger) *OrderService {
	return &OrderService{repo: repo, logger: logger}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	s.logger.Info("creating order",
		zap.Int64("customer_id", req.CustomerID),
		zap.Float64("total_amount", req.TotalAmount),
	)

	order, err := s.repo.Create(ctx, req)
	if err != nil {
		s.logger.Error("failed to create order", zap.Error(err))
		return nil, fmt.Errorf("create order: %w", err)
	}

	s.logger.Info("order created", zap.Int64("order_id", order.ID))
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	s.logger.Info("fetching order", zap.Int64("order_id", id))

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to fetch order", zap.Int64("order_id", id), zap.Error(err))
		return nil, fmt.Errorf("get order: %w", err)
	}

	return order, nil
}
