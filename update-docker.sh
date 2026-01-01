#!/bin/bash

# Script để update Docker image trên server
# Sử dụng: ./update-server.sh [tag]
# Ví dụ: ./update-server.sh dev

set -e

# Cấu hình
IMAGE_NAME="azamiue/go-ingestion"
DEFAULT_TAG="dev"
CONTAINER_NAME="go-ingestion-app"

TAG="${1:-$DEFAULT_TAG}"
FULL_IMAGE_NAME="${IMAGE_NAME}:${TAG}"

echo "=========================================="
echo "Updating Docker container on server"
echo "Image: ${FULL_IMAGE_NAME}"
echo "Container: ${CONTAINER_NAME}"
echo "=========================================="

echo ""
echo "Step 1: Pulling latest image from Docker Hub..."
docker pull "${FULL_IMAGE_NAME}"

if [ $? -eq 0 ]; then
    echo "✓ Pull successful!"
else
    echo "✗ Pull failed!"
    exit 1
fi

echo ""
echo "Step 2: Stopping and removing old container..."
if docker ps -a | grep -q "${CONTAINER_NAME}"; then
    docker stop "${CONTAINER_NAME}" || true
    docker rm "${CONTAINER_NAME}" || true
    echo "✓ Old container removed!"
else
    echo "No existing container found."
fi

echo ""
echo "Step 3: Starting new container..."
docker-compose up -d

if [ $? -eq 0 ]; then
    echo "✓ Container started successfully!"
else
    echo "✗ Failed to start container!"
    exit 1
fi

echo ""
echo "Step 4: Checking container status..."
sleep 2
docker ps | grep "${CONTAINER_NAME}"

echo ""
echo "=========================================="
echo "Update completed successfully!"
echo "=========================================="
echo ""
echo "To view logs, run:"
echo "  docker logs -f ${CONTAINER_NAME}"

