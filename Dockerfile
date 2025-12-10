# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /s3-azure-proxy ./cmd/proxy

# Final stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /s3-azure-proxy /s3-azure-proxy

# Expose port (default 8080)
EXPOSE 8080

# Run as non-root user
USER 65532:65532

ENTRYPOINT ["/s3-azure-proxy"]
