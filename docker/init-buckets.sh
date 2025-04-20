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
create_bucket images
create_bucket videos
create_bucket documents
create_bucket audios
create_bucket other