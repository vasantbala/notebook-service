# --- Build stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /notebook-service ./cmd/notebook-service

# --- Run stage ---
FROM alpine:3.21

# ca-certificates needed for HTTPS calls (JWKS endpoint, OpenAI, rag-anything)
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /notebook-service .
COPY migrations/ migrations/

EXPOSE 8080

ENTRYPOINT ["/app/notebook-service"]
