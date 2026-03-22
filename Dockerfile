# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/tax-module ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/tax-module .
COPY --from=builder /app/.env.example .env.example

EXPOSE 8080

ENTRYPOINT ["./tax-module"]
