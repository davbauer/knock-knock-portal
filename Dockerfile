# Multi-stage Dockerfile for Knock-Knock Portal
# Builds both frontend (SvelteKit) and backend (Go) with minimal final image size

# ============================================================================
# Stage 1: Build Frontend (SvelteKit)
# ============================================================================
FROM node:22-slim AS frontend-builder

WORKDIR /build/frontend

# Enable Corepack for Yarn
RUN corepack enable

# Copy package files and Yarn config
COPY frontend/package.json frontend/yarn.lock frontend/.yarnrc.yml ./

# Install Yarn version from packageManager field and install dependencies
RUN yarn install

# Copy frontend source
COPY frontend/ ./

# Build the frontend (outputs to ../backend/dist_frontend per svelte.config.js)
RUN yarn build

# ============================================================================
# Stage 2: Build Backend (Go)
# ============================================================================
FROM golang:1.24-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build/backend

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY backend/ ./

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -trimpath \
    -o knock-knock \
    ./cmd/server/main.go

# ============================================================================
# Stage 3: Final Runtime Image
# ============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 knockknock && \
    adduser -D -u 1000 -G knockknock knockknock

WORKDIR /app

# Copy binaries and assets
COPY --from=backend-builder /build/backend/knock-knock ./knock-knock
COPY --from=frontend-builder /build/backend/dist_frontend ./dist_frontend

# Create directories
RUN mkdir -p /app/config /app/logs && \
    chown -R knockknock:knockknock /app

USER knockknock

EXPOSE 8000 8080-8099

ENV GIN_MODE=release \
    TZ=UTC

ENTRYPOINT ["/app/knock-knock"]
