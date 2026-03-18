# Build stage for containerized schema generation
# This allows generating Terraform provider schema with a specific minikube version
FROM golang:1.24-alpine

# Build arguments for version control
ARG MINIKUBE_VERSION=v1.37.0

# Install dependencies
RUN apk add --no-cache \
  curl \
  ca-certificates \
  git \
  make

# Download and install the specific minikube version
RUN mkdir -p /usr/local/bin && \
  curl -sSLo /usr/local/bin/minikube \
  "https://github.com/kubernetes/minikube/releases/download/${MINIKUBE_VERSION}/minikube-linux-arm64" && \
  chmod +x /usr/local/bin/minikube && \
  minikube version

# Create workspace directory
WORKDIR /workspace

# Copy project source
COPY . .

# Install Go dependencies
RUN go mod download

# Generate schema
CMD ["make", "schema"]
