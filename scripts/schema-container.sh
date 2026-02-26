#!/bin/bash
#
# Schema generation in a containerized environment
# Allows generating Terraform provider schema with a specific minikube version
#
# Usage: ./scripts/schema-container.sh [MINIKUBE_VERSION]
#        ./scripts/schema-container.sh v1.38.0
#        ./scripts/schema-container.sh          # defaults to v1.37.0

set -e

# Default minikube version if not specified
MINIKUBE_VERSION="${1:-v1.37.0}"

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🐳 Building Docker image with minikube version: $MINIKUBE_VERSION"

# Build Docker image with the specified minikube version
docker build \
    --build-arg "MINIKUBE_VERSION=$MINIKUBE_VERSION" \
    -t terraform-provider-minikube:schema-gen-$MINIKUBE_VERSION \
    -f "$PROJECT_ROOT/Dockerfile" \
    "$PROJECT_ROOT"

echo "✅ Docker image built successfully"
echo "🔨 Generating schema with minikube $MINIKUBE_VERSION..."

# Run the container with workspace mounted
# The volume mount allows the generated file to be available on the host
docker run --rm \
    -v "$PROJECT_ROOT:/workspace" \
    terraform-provider-minikube:schema-gen-$MINIKUBE_VERSION

echo "✅ Schema generation complete"
echo "📝 Generated file: minikube/schema_cluster.go"
echo "💡 Tip: Review changes with 'git diff minikube/schema_cluster.go'"
