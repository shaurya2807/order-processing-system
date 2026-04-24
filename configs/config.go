package configs

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	SQS        SQSConfig
	Redis      RedisConfig
	Storage    StorageConfig
	OpenSearch OpenSearchConfig
}

type OpenSearchConfig struct {
	Endpoint string
}

type SQSConfig struct {
	Region      string
	EndpointURL string
	QueueURL    string
	AccessKey   string
	SecretKey   string
}

type RedisConfig struct {
	Host string
	Port string
}

type StorageConfig struct {
	S3Endpoint   string
	S3Bucket     string
	KMSKeyID     string
	AWSRegion    string
	AWSAccessKey string
	AWSSecretKey string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Load() (*Config, error) {
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "orders"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		SQS: SQSConfig{
			Region:      getEnv("AWS_REGION", "us-east-1"),
			EndpointURL: getEnv("SQS_ENDPOINT", ""),
			QueueURL:    getEnv("SQS_QUEUE_URL", ""),
			AccessKey:   getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretKey:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "localhost"),
			Port: getEnv("REDIS_PORT", "6379"),
		},
		Storage: StorageConfig{
			S3Endpoint:   getEnv("S3_ENDPOINT", ""),
			S3Bucket:     getEnv("S3_BUCKET", "order-artifacts"),
			KMSKeyID:     getEnv("KMS_KEY_ID", ""),
			AWSRegion:    getEnv("AWS_REGION", "us-east-1"),
			AWSAccessKey: getEnv("AWS_ACCESS_KEY_ID", ""),
			AWSSecretKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		OpenSearch: OpenSearchConfig{
			Endpoint: getEnv("OPENSEARCH_ENDPOINT", "http://localhost:9200"),
		},
	}, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
