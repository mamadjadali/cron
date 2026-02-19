# -------------------
# Builder stage
# -------------------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files first → better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary (very small, no libc dependency)
RUN CGO_ENABLED=0 GOOS=linux go build -o /scheduler main.go

# -------------------
# Final tiny image
# -------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Tehran

WORKDIR /app

# Copy only the binary
COPY --from=builder /scheduler .

# Optional: copy config if you later move to env vars / config file
# COPY .env .    # ← uncomment later if needed

# Run the binary
CMD ["/app/scheduler"]