package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/shaurya2807/order-processing-system/internal/model"
	"go.uber.org/zap"
)

type Storage struct {
	client   *s3.Client
	bucket   string
	kmsKeyID string
	logger   *zap.Logger
}

func New(ctx context.Context, endpoint, region, bucket, kmsKeyID, accessKey, secretKey string, logger *zap.Logger) (*Storage, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var clientOpts []func(*s3.Options)
	if endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	return &Storage{
		client:   s3.NewFromConfig(cfg, clientOpts...),
		bucket:   bucket,
		kmsKeyID: kmsKeyID,
		logger:   logger,
	}, nil
}

func (s *Storage) UploadOrder(ctx context.Context, order *model.Order) error {
	s.logger.Info("UploadOrder called", zap.Int64("order_id", order.ID))

	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}

	key := fmt.Sprintf("orders/%d.json", order.ID)

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.bucket),
		Key:                  aws.String(key),
		Body:                 bytes.NewReader(data),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          aws.String("alias/order-artifacts-key"),
	})
	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	s.logger.Info("order uploaded to s3",
		zap.Int64("order_id", order.ID),
		zap.String("bucket", s.bucket),
		zap.String("key", key),
	)

	return nil
}
