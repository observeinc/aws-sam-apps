#!/bin/bash

set -euo pipefail

# ---- CONFIG ----
IMAGE_NAME="${IMAGE_NAME:-aws-sam-apps-all-binaries}"  # Allow override via env
AWS_REGION="${AWS_REGION:-us-west-2}"
ECR_REPO="${ECR_REPO:-723346149663.dkr.ecr.us-west-2.amazonaws.com/aws-sam-apps-orca}"
VERSION="${VERSION:-$(git rev-parse --abbrev-ref HEAD)}"
OS="${OS:-linux}"
ARCH="${ARCH:-arm64}"

echo "🧮 Using:"
echo "  IMAGE_NAME = $IMAGE_NAME"
echo "  VERSION    = $VERSION"
echo "  OS/ARCH    = $OS/$ARCH"
echo "  ECR_REPO   = $ECR_REPO"

# ---- BUILD IMAGE USING MAKE ----
echo "🔧 Building Docker image using Make..."
make docker-build-all-binaries-image IMAGE_NAME=$IMAGE_NAME OS=$OS ARCH=$ARCH VERSION=$VERSION

# ---- LOGIN TO ECR ----
echo "🔐 Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "$ECR_REPO"

# ---- TAG IMAGE ----
echo "🏷️ Tagging image..."
docker tag "$IMAGE_NAME:latest" "$ECR_REPO:latest"

# ---- PUSH IMAGE ----
echo "📦 Pushing image to ECR..."
docker push "$ECR_REPO:latest"

# ----  Print final digest ----
echo "🔍 Final image digest:"
docker inspect --format='{{index .RepoDigests 0}}' "$ECR_REPO:latest" || echo "Digest not available yet"

echo "✅ Done. Image pushed as: $ECR_REPO:latest"
