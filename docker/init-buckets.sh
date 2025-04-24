#!/bin/sh

# Wait for MinIO to start
sleep 10

# Set up MinIO Client alias
mc alias set myminio http://localhost:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"

# Function to create a bucket if it doesn't exist
create_bucket() {
  if ! mc ls myminio/$1 >/dev/null 2>&1; then
    mc mb myminio/$1
    echo "Bucket '$1' created."
  else
    echo "Bucket '$1' already exists."
  fi
}

# Create buckets
create_bucket myapp-public-uploads
create_bucket myapp-temp-uploads

mc anonymous set public myminio/nos3-public-uploads  # Make bucket publicly readable
mc anonymous set none myminio/nos3-temp-uploads      # Make bucket private

mc ilm rule add myminio/nos3-temp-uploads --expiry-days 7
