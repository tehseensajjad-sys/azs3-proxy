# Build stage
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /azs3-proxy ./cmd/proxy

# Final stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /azs3-proxy /azs3-proxy

# Expose port (default 8080)
EXPOSE 8080

# Run as non-root user
USER 65532:65532

ENTRYPOINT ["/azs3-proxy"]
