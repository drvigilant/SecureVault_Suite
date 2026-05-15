# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o securevault .

# Run stage — minimal image
FROM alpine:3.19

RUN addgroup -S vault && adduser -S -G vault vault

WORKDIR /app

COPY --from=builder /build/securevault .
COPY templates/ ./templates/

RUN mkdir -p uploads && chown -R vault:vault /app

USER vault

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:5000 || exit 1

CMD ["./securevault"]
