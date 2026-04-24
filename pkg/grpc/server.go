package orderpb

import (
	"context"

	"github.com/shaurya2807/order-processing-system/internal/model"
	"github.com/shaurya2807/order-processing-system/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderGRPCServer implements OrderServiceServer using the same OrderService
// as the REST handlers so all business logic, caching, and async operations
// are shared between both transports.
type OrderGRPCServer struct {
	UnimplementedOrderServiceServer
	svc *service.OrderService
}

func NewOrderGRPCServer(svc *service.OrderService) *OrderGRPCServer {
	return &OrderGRPCServer{svc: svc}
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error) {
	order, err := s.svc.CreateOrder(ctx, &model.CreateOrderRequest{
		CustomerID:  req.CustomerId,
		TotalAmount: req.TotalAmount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create order: %v", err)
	}
	return orderToResponse(order), nil
}

func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *GetOrderRequest) (*OrderResponse, error) {
	order, err := s.svc.GetOrder(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "order %d not found", req.Id)
	}
	return orderToResponse(order), nil
}

func orderToResponse(o *model.Order) *OrderResponse {
	return &OrderResponse{
		Id:          o.ID,
		CustomerId:  o.CustomerID,
		Status:      string(o.Status),
		TotalAmount: o.TotalAmount,
		CreatedAt:   o.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
