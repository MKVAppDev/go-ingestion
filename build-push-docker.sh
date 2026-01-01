#!/bin/bash

# Script để build và push Docker image lên Docker Hub với multi-platform support
# Sử dụng: ./build-and-push.sh [tag]
# Ví dụ: ./build-and-push.sh dev

set -e

# Cấu hình
IMAGE_NAME="azamiue/go-ingestion"
DEFAULT_TAG="dev"
PLATFORMS="linux/amd64,linux/arm64"

TAG="${1:-$DEFAULT_TAG}"
FULL_IMAGE_NAME="${IMAGE_NAME}:${TAG}"

echo "=========================================="
echo "Building and pushing Docker image"
echo "Image: ${FULL_IMAGE_NAME}"
echo "Platforms: ${PLATFORMS}"
echo "=========================================="

echo ""
echo "Step 1: Setting up Docker buildx..."
if ! docker buildx ls | grep -q "multiplatform"; then
    echo "Creating new buildx builder 'multiplatform'..."
    docker buildx create --name multiplatform --use
    docker buildx inspect --bootstrap
else
    echo "Using existing buildx builder 'multiplatform'..."
    docker buildx use multiplatform
fi

if [ $? -eq 0 ]; then
    echo "✓ Buildx setup successful!"
else
    echo "✗ Buildx setup failed!"
    exit 1
fi

echo ""
echo "Step 2: Checking Docker Hub login..."
if ! docker info | grep -q "Username"; then
    echo "Please login to Docker Hub:"
    docker login
fi

echo ""
echo "Step 3: Building and pushing multi-platform image..."
echo "This may take a while..."
docker buildx build \
    --platform "${PLATFORMS}" \
    --tag "${FULL_IMAGE_NAME}" \
    --push \
    .

if [ $? -eq 0 ]; then
    echo "✓ Build and push successful!"
    echo ""
    echo "=========================================="
    echo "Multi-platform image has been pushed!"
    echo "Image: ${FULL_IMAGE_NAME}"
    echo "Platforms: ${PLATFORMS}"
    echo "=========================================="
else
    echo "✗ Build and push failed!"
    exit 1
fi

echo ""
echo "Verifying image manifest..."
docker buildx imagetools inspect "${FULL_IMAGE_NAME}"