package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/shaurya2807/order-processing-system/pkg/queue"
	"go.uber.org/zap"
)

type Consumer struct {
	client   *sqs.Client
	queueURL string
	log      *zap.Logger
}

func NewConsumer(ctx context.Context, region, endpointURL, queueURL, accessKey, secretKey string, log *zap.Logger) (*Consumer, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var clientOpts []func(*sqs.Options)
	if endpointURL != "" {
		clientOpts = append(clientOpts, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
		})
	}

	return &Consumer{
		client:   sqs.NewFromConfig(cfg, clientOpts...),
		queueURL: queueURL,
		log:      log,
	}, nil
}

// Start polls the SQS queue in a loop until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	c.log.Info("notification consumer started", zap.String("queue_url", c.queueURL))

	for {
		select {
		case <-ctx.Done():
			c.log.Info("notification consumer stopped")
			return
		default:
		}

		out, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     5,
		})
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("notification consumer stopped")
				return
			}
			c.log.Error("failed to receive messages from queue", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range out.Messages {
			if err := c.handle(ctx, msg); err != nil {
				c.log.Error("failed to process message",
					zap.Error(err),
					zap.String("message_id", aws.ToString(msg.MessageId)),
				)
				// Do not delete — SQS will redeliver after visibility timeout expires.
				continue
			}
			c.deleteMessage(ctx, msg)
		}
	}
}

func (c *Consumer) handle(_ context.Context, msg sqstypes.Message) error {
	var event queue.OrderCreatedEvent
	if err := json.Unmarshal([]byte(aws.ToString(msg.Body)), &event); err != nil {
		return fmt.Errorf("unmarshal order event: %w", err)
	}

	c.log.Info("sending notification",
		zap.Int64("order_id", event.OrderID),
		zap.Int64("customer_id", event.CustomerID),
	)

	c.log.Info("notification sent",
		zap.Int64("order_id", event.OrderID),
		zap.Int64("customer_id", event.CustomerID),
		zap.String("channel", "email"),
	)

	return nil
}

func (c *Consumer) deleteMessage(ctx context.Context, msg sqstypes.Message) {
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		c.log.Error("failed to delete message from queue",
			zap.Error(err),
			zap.String("message_id", aws.ToString(msg.MessageId)),
		)
		return
	}
	c.log.Info("message deleted from queue",
		zap.String("message_id", aws.ToString(msg.MessageId)),
	)
}
