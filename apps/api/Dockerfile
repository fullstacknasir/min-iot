# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api ./cmd/api/main.go

# Production stage
FROM alpine:3.19

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /api /app/api

EXPOSE 8080

CMD ["/app/api"]
