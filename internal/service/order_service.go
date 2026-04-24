package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shaurya2807/order-processing-system/internal/model"
	"github.com/shaurya2807/order-processing-system/internal/repository"
	"github.com/shaurya2807/order-processing-system/pkg/cache"
	"github.com/shaurya2807/order-processing-system/pkg/queue"
	"go.uber.org/zap"
)

type OrderService struct {
	repo      *repository.OrderRepository
	publisher *queue.Publisher
	cache     *cache.Cache
	logger    *zap.Logger
}

func NewOrderService(repo *repository.OrderRepository, publisher *queue.Publisher, cache *cache.Cache, logger *zap.Logger) *OrderService {
	return &OrderService{repo: repo, publisher: publisher, cache: cache, logger: logger}
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

	key := orderCacheKey(order.ID)
	if delErr := s.cache.Delete(ctx, key); delErr != nil {
		s.logger.Error("cache delete error", zap.String("key", key), zap.Error(delErr))
	}

	event := queue.OrderCreatedEvent{
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		TotalAmount: order.TotalAmount,
		Status:      string(order.Status),
		CreatedAt:   order.CreatedAt,
	}
	if pubErr := s.publisher.PublishOrderCreated(ctx, event); pubErr != nil {
		s.logger.Error("failed to publish order created event",
			zap.Int64("order_id", order.ID),
			zap.Error(pubErr),
		)
	} else {
		s.logger.Info("order created event published", zap.Int64("order_id", order.ID))
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	s.logger.Info("fetching order", zap.Int64("order_id", id))

	key := orderCacheKey(id)

	cached, err := s.cache.Get(ctx, key)
	if err != nil {
		s.logger.Error("cache get error", zap.String("key", key), zap.Error(err))
	} else if cached != "" {
		s.logger.Info("cache hit", zap.String("key", key))
		var order model.Order
		if jsonErr := json.Unmarshal([]byte(cached), &order); jsonErr == nil {
			return &order, nil
		}
		s.logger.Error("failed to unmarshal cached order", zap.String("key", key), zap.Error(err))
	} else {
		s.logger.Info("cache miss", zap.String("key", key))
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to fetch order", zap.Int64("order_id", id), zap.Error(err))
		return nil, fmt.Errorf("get order: %w", err)
	}

	if data, marshalErr := json.Marshal(order); marshalErr == nil {
		if setErr := s.cache.Set(ctx, key, string(data)); setErr != nil {
			s.logger.Error("cache set error", zap.String("key", key), zap.Error(setErr))
		}
	}

	return order, nil
}

func orderCacheKey(id int64) string {
	return fmt.Sprintf("order:%d", id)
}
