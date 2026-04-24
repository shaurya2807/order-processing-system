#!/bin/bash
awslocal sqs create-queue --queue-name orders-queue

awslocal s3 mb s3://order-artifacts

KMS_KEY_ID=$(awslocal kms create-key --description "order-artifacts-key" --query 'KeyMetadata.KeyId' --output text)
echo "Created KMS key: $KMS_KEY_ID"
awslocal kms create-alias --alias-name alias/order-artifacts-key --target-key-id "$KMS_KEY_ID"
