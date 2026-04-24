package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/shaurya2807/order-processing-system/internal/model"
	"go.uber.org/zap"
)

const ordersIndex = "orders"

type Client struct {
	os     *opensearch.Client
	logger *zap.Logger
}

func New(endpoint string, logger *zap.Logger) (*Client, error) {
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{endpoint},
		Username:  "admin",
		Password:  "admin",
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch client: %w", err)
	}
	return &Client{os: client, logger: logger}, nil
}

type orderDoc struct {
	OrderID     int64     `json:"order_id"`
	CustomerID  int64     `json:"customer_id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
	IndexedAt   time.Time `json:"indexed_at"`
}

func (c *Client) IndexOrder(ctx context.Context, order *model.Order) error {
	doc := orderDoc{
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		Status:      string(order.Status),
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
		IndexedAt:   time.Now().UTC(),
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal order doc: %w", err)
	}

	req := opensearchapi.IndexRequest{
		Index:      ordersIndex,
		DocumentID: fmt.Sprintf("%d", order.ID),
		Body:       bytes.NewReader(body),
	}

	res, err := req.Do(ctx, c.os)
	if err != nil {
		return fmt.Errorf("opensearch index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("opensearch index error: %s", res.String())
	}

	c.logger.Info("order indexed in opensearch",
		zap.Int64("order_id", order.ID),
		zap.String("index", ordersIndex),
	)
	return nil
}
